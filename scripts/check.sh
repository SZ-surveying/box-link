#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROFILE="${BOX_LINK_CHECK_PROFILE:-full}"

RUN_LINT=1
RUN_TESTS=1
RUN_BUILD=1
RUN_MOD_TIDY=1

export GOCACHE="${GOCACHE:-$ROOT_DIR/.gocache}"
export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-$ROOT_DIR/.cache/golangci-lint}"

usage() {
  cat <<'EOF'
Usage: ./scripts/check.sh [--profile pre-commit|ci|full] [--pre-commit] [--ci] [--full] [--skip-lint] [--skip-tests] [--skip-build] [--skip-mod-tidy]
EOF
}

log_step() {
  printf '\n[check] %s\n' "$1"
}

fail() {
  printf '[check] %s\n' "$1" >&2
  exit 1
}

run_lint() {
  if command -v golangci-lint >/dev/null 2>&1; then
    log_step "Running golangci-lint via local toolchain"
    golangci-lint run --config=.golangci.yml ./...
    return
  fi

  if command -v devbox >/dev/null 2>&1; then
    log_step "Running golangci-lint via devbox"
    devbox run -- golangci-lint run --config=.golangci.yml ./...
    return
  fi

  fail "golangci-lint is required; install it locally or use devbox"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      [[ $# -ge 2 ]] || {
        usage >&2
        exit 1
      }
      PROFILE="$2"
      shift 2
      ;;
    --pre-commit)
      PROFILE="pre-commit"
      shift
      ;;
    --ci)
      PROFILE="ci"
      shift
      ;;
    --full)
      PROFILE="full"
      shift
      ;;
    --skip-lint)
      RUN_LINT=0
      shift
      ;;
    --skip-tests)
      RUN_TESTS=0
      shift
      ;;
    --skip-build)
      RUN_BUILD=0
      shift
      ;;
    --skip-mod-tidy)
      RUN_MOD_TIDY=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

case "$PROFILE" in
  pre-commit)
    RUN_BUILD=0
    RUN_MOD_TIDY=0
    ;;
  ci)
    RUN_MOD_TIDY=0
    ;;
  full)
    ;;
  *)
    fail "unknown profile: $PROFILE"
    ;;
esac

cd "$ROOT_DIR"
mkdir -p "$GOCACHE" "$GOLANGCI_LINT_CACHE"

if [[ "$RUN_LINT" -eq 1 ]]; then
  run_lint
fi

if [[ "$RUN_TESTS" -eq 1 ]]; then
  log_step "Running go test"
  go test ./...
fi

if [[ "$RUN_BUILD" -eq 1 ]]; then
  log_step "Running go build"
  go build ./...
fi

if [[ "$RUN_MOD_TIDY" -eq 1 ]]; then
  log_step "Verifying go.mod and go.sum stay tidy"
  cp go.mod go.mod.bak
  cp go.sum go.sum.bak
  trap 'mv -f go.mod.bak go.mod 2>/dev/null || true; mv -f go.sum.bak go.sum 2>/dev/null || true' EXIT
  go mod tidy
  if ! git diff --exit-code -- go.mod go.sum >/dev/null; then
    git diff -- go.mod go.sum >&2 || true
    fail "go.mod or go.sum changed after go mod tidy"
  fi
  rm -f go.mod.bak go.sum.bak
  trap - EXIT
fi

log_step "Checks passed"

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCOPE="staged"
VERIFY=0

usage() {
  cat <<'EOF'
Usage: ./scripts/format.sh [--scope staged|modified|all] [--check]

Options:
  --scope <value>  Choose which files to inspect
  --check          Verify formatting without changing files
EOF
}

log_step() {
  printf '\n[format] %s\n' "$1"
}

list_go_files() {
  case "$SCOPE" in
    staged)
      git -C "$ROOT_DIR" diff --cached --name-only --diff-filter=ACMR
      ;;
    modified)
      {
        git -C "$ROOT_DIR" diff --cached --name-only --diff-filter=ACMR
        git -C "$ROOT_DIR" diff --name-only --diff-filter=ACMR
        git -C "$ROOT_DIR" ls-files --others --exclude-standard
      } | awk '!seen[$0]++'
      ;;
    all)
      find "$ROOT_DIR/cmd" "$ROOT_DIR/internal" -type f -name '*.go' -print | sed "s#^$ROOT_DIR/##"
      ;;
    *)
      echo "[format] unknown scope: $SCOPE" >&2
      exit 1
      ;;
  esac | grep -E '^(cmd|internal)/.*\.go$' || true
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --scope)
      [[ $# -ge 2 ]] || {
        usage >&2
        exit 1
      }
      SCOPE="$2"
      shift 2
      ;;
    --check)
      VERIFY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "[format] unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

cd "$ROOT_DIR"
mapfile -t GO_FILES < <(list_go_files)

if [[ "${#GO_FILES[@]}" -eq 0 ]]; then
  log_step "No Go files matched scope=$SCOPE"
  exit 0
fi

if [[ "$VERIFY" -eq 1 ]]; then
  log_step "Checking Go formatting for ${#GO_FILES[@]} file(s)"
  mapfile -t UNFORMATTED < <(gofmt -l "${GO_FILES[@]}")
  if [[ "${#UNFORMATTED[@]}" -gt 0 ]]; then
    printf '[format] Unformatted files:\n' >&2
    printf '  - %s\n' "${UNFORMATTED[@]}" >&2
    exit 1
  fi
  log_step "Formatting check passed"
  exit 0
fi

log_step "Formatting ${#GO_FILES[@]} Go file(s)"
gofmt -w "${GO_FILES[@]}"
log_step "Formatting completed"

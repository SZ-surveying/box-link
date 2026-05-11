#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
README_PATH="$ROOT_DIR/README.md"
INSTALL_SCRIPT="$ROOT_DIR/packaging/install.sh"
RELEASE_MODE=0
VERSION=""
CHECKSUM_TOOL=()

usage() {
  cat <<'EOF'
Usage:
  ./packaging/package.sh [--version <value>]
  ./packaging/package.sh --release [--version <value>]

Options:
  --version <value>  Version label used in artifact names
  --release          Build the release target set: darwin/arm64, darwin/amd64
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      if [[ -z "$VERSION" ]]; then
        echo "--version requires a value" >&2
        exit 1
      fi
      shift 2
      ;;
    --release)
      RELEASE_MODE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$ROOT_DIR" describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

if command -v shasum >/dev/null 2>&1; then
  CHECKSUM_TOOL=(shasum -a 256)
elif command -v sha256sum >/dev/null 2>&1; then
  CHECKSUM_TOOL=(sha256sum)
else
  echo "could not find shasum or sha256sum" >&2
  exit 1
fi

cd "$ROOT_DIR"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

build_target() {
  local goos="$1"
  local goarch="$2"
  local out_dir="$DIST_DIR/box-link-$VERSION-$goos-$goarch"

  mkdir -p "$out_dir"

  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build \
      -trimpath \
      -ldflags="-s -w -X main.version=$VERSION" \
      -o "$out_dir/box-link" \
      ./cmd/box-link

  cp "$README_PATH" "$out_dir/README.md"
  cp "$INSTALL_SCRIPT" "$out_dir/install.sh"
  chmod +x "$out_dir/install.sh"

  tar -C "$DIST_DIR" -czf "$DIST_DIR/box-link-$VERSION-$goos-$goarch.tar.gz" "$(basename "$out_dir")"
}

if [[ "$RELEASE_MODE" -eq 1 ]]; then
  build_target darwin arm64
  build_target darwin amd64
else
  build_target "$(go env GOOS)" "$(go env GOARCH)"
fi

"${CHECKSUM_TOOL[@]}" "$DIST_DIR"/*.tar.gz > "$DIST_DIR/checksums.txt"
printf 'artifacts written to %s\n' "$DIST_DIR"

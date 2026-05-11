#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
VERSION=""
IDENTIFIER="com.example.box-link"
INSTALL_LOCATION="/"
PKG_NAME=""
OUTPUT_PATH=""

usage() {
  cat <<'EOF'
Usage:
  ./packaging/build-pkg.sh [--version <value>] [--identifier <value>] [--output <path>]

Options:
  --version <value>     Version label used for the package name and embedded binary
  --identifier <value>  macOS package identifier, default: com.example.box-link
  --output <path>       Output `.pkg` path, default: dist/box-link-<version>.pkg
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      [[ -n "$VERSION" ]] || {
        echo "--version requires a value" >&2
        exit 1
      }
      shift 2
      ;;
    --identifier)
      IDENTIFIER="${2:-}"
      [[ -n "$IDENTIFIER" ]] || {
        echo "--identifier requires a value" >&2
        exit 1
      }
      shift 2
      ;;
    --output)
      OUTPUT_PATH="${2:-}"
      [[ -n "$OUTPUT_PATH" ]] || {
        echo "--output requires a value" >&2
        exit 1
      }
      shift 2
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

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "build-pkg.sh only works on macOS because it uses pkgbuild" >&2
  exit 1
fi

if ! command -v pkgbuild >/dev/null 2>&1; then
  echo "pkgbuild is required to produce a .pkg installer" >&2
  exit 1
fi

if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$ROOT_DIR" describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

if [[ -z "$OUTPUT_PATH" ]]; then
  OUTPUT_PATH="$DIST_DIR/box-link-$VERSION.pkg"
fi

cd "$ROOT_DIR"
./packaging/package.sh --version "$VERSION"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"
PACKAGE_DIR="$DIST_DIR/box-link-$VERSION-$GOOS-$GOARCH"
BINARY_PATH="$PACKAGE_DIR/box-link"

if [[ ! -x "$BINARY_PATH" ]]; then
  echo "expected packaged binary at $BINARY_PATH" >&2
  exit 1
fi

PKG_STAGE_DIR="$DIST_DIR/pkg-stage"
rm -rf "$PKG_STAGE_DIR"
mkdir -p "$PKG_STAGE_DIR/usr/local/bin"
cp "$BINARY_PATH" "$PKG_STAGE_DIR/usr/local/bin/box-link"

mkdir -p "$(dirname "$OUTPUT_PATH")"

pkgbuild \
  --root "$PKG_STAGE_DIR" \
  --identifier "$IDENTIFIER" \
  --version "${VERSION#v}" \
  --install-location "$INSTALL_LOCATION" \
  "$OUTPUT_PATH"

printf 'pkg written to %s\n' "$OUTPUT_PATH"

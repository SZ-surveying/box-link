#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
VERSION=""
REPO="your-org/box-link"
OUTPUT_PATH=""
CHECKSUM_TOOL=()
FORMULA_NAME=""
TAP_DIR=""
FORMULA_CLASS=""

usage() {
  cat <<'EOF'
Usage:
  ./packaging/generate-homebrew-formula.sh [--version <value>] [--repo <owner/repo>] [--output <path>]
  ./packaging/generate-homebrew-formula.sh [--version <value>] [--repo <owner/repo>] [--tap-dir <path>] [--formula-name <name>]

Options:
  --version <value>  Version label used in release asset names
  --repo <value>     GitHub repository slug, default: your-org/box-link
  --output <path>    Formula output path, default: packaging/homebrew/box-link.rb
  --tap-dir <path>   Write Formula/<name>.rb into a Homebrew tap repository
  --formula-name     Formula file stem, default: box-link
EOF
}

formula_class_name() {
  local input="$1"
  local token=""
  local lower=""
  local out=""

  input="${input//[^[:alnum:]]/ }"
  for token in $input; do
    lower="${token,,}"
    out+="${lower^}"
  done

  if [[ -z "$out" ]]; then
    echo "could not derive a Homebrew class name from formula name" >&2
    exit 1
  fi

  printf '%s\n' "$out"
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
    --repo)
      REPO="${2:-}"
      [[ -n "$REPO" ]] || {
        echo "--repo requires a value" >&2
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
    --tap-dir)
      TAP_DIR="${2:-}"
      [[ -n "$TAP_DIR" ]] || {
        echo "--tap-dir requires a value" >&2
        exit 1
      }
      shift 2
      ;;
    --formula-name)
      FORMULA_NAME="${2:-}"
      [[ -n "$FORMULA_NAME" ]] || {
        echo "--formula-name requires a value" >&2
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

if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$ROOT_DIR" describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

if [[ -z "$FORMULA_NAME" ]]; then
  FORMULA_NAME="${REPO##*/}"
fi

if [[ -n "$TAP_DIR" ]]; then
  OUTPUT_PATH="$TAP_DIR/Formula/$FORMULA_NAME.rb"
elif [[ -z "$OUTPUT_PATH" ]]; then
  OUTPUT_PATH="$ROOT_DIR/packaging/homebrew/$FORMULA_NAME.rb"
fi

FORMULA_CLASS="$(formula_class_name "$FORMULA_NAME")"

if command -v shasum >/dev/null 2>&1; then
  CHECKSUM_TOOL=(shasum -a 256)
elif command -v sha256sum >/dev/null 2>&1; then
  CHECKSUM_TOOL=(sha256sum)
else
  echo "could not find shasum or sha256sum" >&2
  exit 1
fi

cd "$ROOT_DIR"
./packaging/package.sh --release --version "$VERSION"

arm64_archive="$DIST_DIR/box-link-$VERSION-darwin-arm64.tar.gz"
amd64_archive="$DIST_DIR/box-link-$VERSION-darwin-amd64.tar.gz"

if [[ ! -f "$arm64_archive" || ! -f "$amd64_archive" ]]; then
  echo "expected release archives in $DIST_DIR" >&2
  exit 1
fi

arm64_sha="$("${CHECKSUM_TOOL[@]}" "$arm64_archive" | awk '{print $1}')"
amd64_sha="$("${CHECKSUM_TOOL[@]}" "$amd64_archive" | awk '{print $1}')"
normalized_version="${VERSION#v}"

mkdir -p "$(dirname "$OUTPUT_PATH")"

cat >"$OUTPUT_PATH" <<EOF
class $FORMULA_CLASS < Formula
  desc "Direct-link box networking tool for macOS"
  homepage "https://github.com/$REPO"
  version "$normalized_version"

  on_arm do
    url "https://github.com/$REPO/releases/download/v#{version}/box-link-v#{version}-darwin-arm64.tar.gz"
    sha256 "$arm64_sha"
  end

  on_intel do
    url "https://github.com/$REPO/releases/download/v#{version}/box-link-v#{version}-darwin-amd64.tar.gz"
    sha256 "$amd64_sha"
  end

  def install
    bin.install "box-link"
    prefix.install "README.md"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/box-link version")
  end
end
EOF

printf 'formula written to %s\n' "$OUTPUT_PATH"

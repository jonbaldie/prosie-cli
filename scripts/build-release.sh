#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

VERSION="${1:-${VERSION:-}}"
if [ -z "$VERSION" ]; then
  if [ -z "$(git -C "$ROOT_DIR" status --porcelain --untracked-files=no 2>/dev/null)" ]; then
    VERSION=$(git -C "$ROOT_DIR" describe --exact-match --tags --match 'v[0-9]*' HEAD 2>/dev/null || true)
  fi
fi
if [ -z "$VERSION" ]; then
  VERSION=$(grep -E 'fallbackVersion[[:space:]]*=' "${ROOT_DIR}/internal/version/version.go" | sed -E 's/.*"([^"]+)".*/\1/')
fi
VERSION="${VERSION#v}"

DIST_DIR="${DIST_DIR:-${ROOT_DIR}/dist}"
mkdir -p "$DIST_DIR"

# Clean prior release artifacts from dist directory
rm -f "${DIST_DIR}"/prosie_* "${DIST_DIR}"/checksums.txt

PLATFORMS=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

echo "Building prosie-cli release v${VERSION} to ${DIST_DIR}..."

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"
  echo "==> Building ${GOOS}/${GOARCH}..."

  BINARY_NAME="prosie"
  if [ "$GOOS" = "windows" ]; then
    BINARY_NAME="prosie.exe"
  fi

  BUILD_TMP="$(mktemp -d)"

  (
    cd "$ROOT_DIR"
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
      -trimpath \
      -ldflags="-s -w -X github.com/jonbaldie/prosie-cli/internal/version.Version=${VERSION}" \
      -o "$BUILD_TMP/$BINARY_NAME" \
      .
  )

  cp "${ROOT_DIR}/LICENSE" "$BUILD_TMP/"
  if [ -f "${ROOT_DIR}/README.md" ]; then
    cp "${ROOT_DIR}/README.md" "$BUILD_TMP/"
  fi

  FILES=("$BINARY_NAME" "LICENSE")
  if [ -f "$BUILD_TMP/README.md" ]; then
    FILES+=("README.md")
  fi

  ARCHIVE_BASE="prosie_${VERSION}_${GOOS}_${GOARCH}"

  if [ "$GOOS" = "windows" ]; then
    ARCHIVE_FILE="${ARCHIVE_BASE}.zip"
    (cd "$BUILD_TMP" && zip -q -r "${DIST_DIR}/${ARCHIVE_FILE}" "${FILES[@]}")
  else
    ARCHIVE_FILE="${ARCHIVE_BASE}.tar.gz"
    tar -czf "${DIST_DIR}/${ARCHIVE_FILE}" -C "$BUILD_TMP" "${FILES[@]}"
  fi

  rm -rf "$BUILD_TMP"
done

echo "==> Generating checksums.txt..."
(
  cd "$DIST_DIR"
  rm -f checksums.txt
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum prosie_* > checksums.txt
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 prosie_* > checksums.txt
  else
    echo "Error: neither sha256sum nor shasum found" >&2
    exit 1
  fi
)

echo "==> Release build complete:"
cat "${DIST_DIR}/checksums.txt"

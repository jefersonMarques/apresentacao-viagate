#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

VERSION="${VERSION:-$(git rev-parse --short=12 HEAD)}"
GOOS_VALUE="${GOOS:-linux}"
GOARCH_VALUE="${GOARCH:-amd64}"
SKIP_CHECKS="${SKIP_CHECKS:-0}"
DIST_ROOT="$ROOT_DIR/dist"
RELEASE_NAME="viagate-commercial-${VERSION}-${GOOS_VALUE}-${GOARCH_VALUE}"
RELEASE_DIR="$DIST_ROOT/$RELEASE_NAME"
ARCHIVE_PATH="$DIST_ROOT/$RELEASE_NAME.tar.gz"
CHECKSUM_PATH="$ARCHIVE_PATH.sha256"

if [[ "$GOOS_VALUE" != "linux" ]]; then
  echo "build-release.sh currently packages the production runtime for Linux only" >&2
  exit 2
fi

if [[ "$SKIP_CHECKS" != "1" ]]; then
  go run github.com/a-h/templ/cmd/templ@v0.3.943 generate
  go vet ./...
  go test ./...
fi

rm -rf "$RELEASE_DIR" "$ARCHIVE_PATH" "$CHECKSUM_PATH"
mkdir -p "$RELEASE_DIR"

BUILD_FLAGS=(-trimpath -ldflags="-s -w")
CGO_ENABLED=0 GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" go build "${BUILD_FLAGS[@]}" -o "$RELEASE_DIR/server" ./cmd/server
CGO_ENABLED=0 GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" go build "${BUILD_FLAGS[@]}" -o "$RELEASE_DIR/migrate" ./cmd/migrate
CGO_ENABLED=0 GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" go build "${BUILD_FLAGS[@]}" -o "$RELEASE_DIR/preflight" ./cmd/preflight

cp -R migrations "$RELEASE_DIR/migrations"
cp -R assets "$RELEASE_DIR/assets"
cp -R proposal "$RELEASE_DIR/proposal"
mkdir -p "$RELEASE_DIR/web"
cp -R web/assets "$RELEASE_DIR/web/assets"
cp presentation-content.html "$RELEASE_DIR/presentation-content.html"

for file in ./*.css ./*.js; do
  [[ -f "$file" ]] || continue
  cp "$file" "$RELEASE_DIR/"
done

cp README.md LICENSE "$RELEASE_DIR/"
printf '%s\n' "$VERSION" > "$RELEASE_DIR/VERSION"
printf '%s\n' "$(git rev-parse HEAD)" > "$RELEASE_DIR/GIT_SHA"

tar -C "$DIST_ROOT" -czf "$ARCHIVE_PATH" "$RELEASE_NAME"
(
  cd "$DIST_ROOT"
  sha256sum "$(basename "$ARCHIVE_PATH")" > "$(basename "$CHECKSUM_PATH")"
)

echo "release: $ARCHIVE_PATH"
echo "checksum: $CHECKSUM_PATH"

#!/usr/bin/env bash
# Cross-compiles the aicontained binary for every supported platform.
# Pure Go cross-compilation (CGO disabled) — no platform toolchains needed.
# Output lands in dist/, gitignored.
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
for platform in windows/amd64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os=${platform%/*} arch=${platform#*/}
  out="dist/aicontained-$os-$arch"
  [ "$os" = windows ] && out+=".exe"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -o "$out" ./cmd/aicontained
done
ls -lh dist

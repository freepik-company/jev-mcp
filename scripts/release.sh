#!/bin/sh
set -eu
out_dir=${1:-/dist}
mkdir -p "$out_dir"
for target_os in linux darwin; do
    for target_arch in amd64 arm64; do
        staging_dir=$(mktemp -d)
        CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
            go build -trimpath -ldflags='-s -w' -o "$staging_dir/jev-mcp" ./cmd/jev-mcp
        cp LICENSE "$staging_dir/LICENSE"
        tar -czf "$out_dir/jev-mcp_${target_os}_${target_arch}.tar.gz" -C "$staging_dir" jev-mcp LICENSE
        rm -rf "$staging_dir"
    done
done
cd "$out_dir"
sha256sum jev-mcp_*.tar.gz > checksums.txt

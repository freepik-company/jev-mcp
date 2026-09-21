#!/bin/sh
# Builds the release archives for every supported platform and their checksums.
# Usage: release.sh [output_dir] ; VERSION selects the reported server version.
set -eu
out_dir=${1:-/dist}
version=${VERSION:-dev}
mkdir -p "$out_dir"
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
    target_os=${target%/*}
    target_arch=${target#*/}
    binary=jev-mcp
    [ "$target_os" = windows ] && binary=jev-mcp.exe
    staging_dir=$(mktemp -d)
    CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
        go build -trimpath \
        -ldflags="-s -w -X github.com/freepik-company/jev-mcp/internal/mcp.Version=${version}" \
        -o "$staging_dir/$binary" ./cmd/jev-mcp
    cp LICENSE "$staging_dir/LICENSE"
    tar -czf "$out_dir/jev-mcp_${target_os}_${target_arch}.tar.gz" -C "$staging_dir" "$binary" LICENSE
    rm -rf "$staging_dir"
done
cd "$out_dir"
sha256sum jev-mcp_*.tar.gz > checksums.txt

FROM golang:1.26-trixie AS source
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM source AS test
CMD ["go", "test", "-race", "./..."]

FROM source AS build
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/freepik-company/jev-mcp/internal/mcp.Version=${VERSION}" \
    -o /out/jev-mcp ./cmd/jev-mcp

FROM debian:trixie-slim AS runtime
ARG VERSION=dev
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/jev-mcp /usr/local/bin/jev-mcp
COPY LICENSE /usr/share/licenses/jev-mcp/LICENSE
# The MCP Registry verifies image ownership through this label; it must match server.json.
LABEL io.modelcontextprotocol.server.name="io.github.freepik-company/jev-mcp" \
      org.opencontainers.image.title="jev-mcp" \
      org.opencontainers.image.description="MCP server for typed decisions with Jev / System One via OpenRouter or TypeSafe" \
      org.opencontainers.image.source="https://github.com/freepik-company/jev-mcp" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}"
USER 65532:65532
ENTRYPOINT ["jev-mcp"]

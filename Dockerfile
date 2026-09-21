FROM golang:1.26-trixie AS source
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM source AS test
CMD ["go", "test", "-race", "./..."]

FROM source AS build
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/jev-mcp ./cmd/jev-mcp

FROM debian:trixie-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/jev-mcp /usr/local/bin/jev-mcp
COPY LICENSE /usr/share/licenses/jev-mcp/LICENSE
USER 65532:65532
ENTRYPOINT ["jev-mcp"]

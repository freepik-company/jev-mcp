.PHONY: test image release

IMAGE ?= jev-mcp:local
# Release builds pass the tag (v1.2.3 -> 1.2.3); local builds report "dev".
VERSION ?= dev

test:
	docker build --target test -t $(IMAGE)-test .
	docker run --rm $(IMAGE)-test sh -c 'test -z "$$(gofmt -l .)" && go vet ./... && go test -race ./...'

image:
	docker build --target runtime --build-arg VERSION=$(VERSION) -t $(IMAGE) .

# Artifacts are built inside a container; Go is not required on the host.
release:
	docker build --target source -t $(IMAGE)-source .
	mkdir -p dist
	docker run --rm -e VERSION=$(VERSION) -v "$(CURDIR)/dist:/dist" $(IMAGE)-source sh scripts/release.sh

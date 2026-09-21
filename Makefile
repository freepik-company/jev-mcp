.PHONY: test image release

IMAGE ?= jev-mcp:local

test:
	docker build --target test -t $(IMAGE)-test .
	docker run --rm $(IMAGE)-test sh -c 'test -z "$$(gofmt -l .)" && go vet ./... && go test -race ./...'

image:
	docker build --target runtime -t $(IMAGE) .

# Los artefactos salen de un contenedor; no hace falta Go en el host.
release:
	docker build --target source -t $(IMAGE)-source .
	mkdir -p dist
	docker run --rm -v "$(CURDIR)/dist:/dist" $(IMAGE)-source sh scripts/release.sh

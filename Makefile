SHELL := /bin/sh

BINARY := bin/gateway
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/fijimunkii/chatgpt-plugin-tailscale/internal/version.Version=$(VERSION) \
	-X github.com/fijimunkii/chatgpt-plugin-tailscale/internal/version.Commit=$(COMMIT) \
	-X github.com/fijimunkii/chatgpt-plugin-tailscale/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: all fmt-check scripts-check test vet build check run up operator-up bootstrap bootstrap-check docker clean

all: check

fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "Go files require gofmt:"; gofmt -l .; exit 1; }

scripts-check:
	bash -n bootstrap scripts/*.sh
	./scripts/test-bootstrap.sh

test:
	go test ./...

vet:
	go vet ./...

build:
	@mkdir -p bin
	go build -trimpath -ldflags='$(LDFLAGS)' -o $(BINARY) ./cmd/gateway

check: scripts-check fmt-check test vet build

run:
	go run ./cmd/gateway

up:
	./scripts/run-local.sh

operator-up:
	OPERATOR_MODE=true ./scripts/run-local.sh

bootstrap:
	./bootstrap

bootstrap-check:
	./bootstrap --check

docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t chatgpt-plugin-tailscale:$(VERSION) .

clean:
	rm -rf bin coverage.out

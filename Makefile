.PHONY: build build-server run-server restart kill proto test test-arango test-all vet lint clean

export PATH := /usr/local/go/bin:$(PATH)

# ── Build ─────────────────────────────────────────────────────────────────────

## Verify the module compiles cleanly.
build:
	go build ./...

## Build the service binary to bin/codevaldorg.
build-server:
	go build -o bin/codevaldorg ./cmd

## Build and run the service.
## ArangoDB, Cross, and OAuth vars can be placed in a .env file (loaded automatically).
run-server: build-server
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a; \
	fi; \
	./bin/codevaldorg

## Stop any running instance, rebuild, and run.
restart: kill build-server
	@echo "Running codevaldorg..."
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a; \
	fi; \
	./bin/codevaldorg

## Stop any running instances of codevaldorg.
kill:
	@echo "Stopping any running instances..."
	-@pkill -x codevaldorg 2>/dev/null || true
	-@ADDR="$${BIND_ADDR:-:9090}"; fuser -k "$${ADDR##*:}/tcp" 2>/dev/null || true
	@sleep 1

# ── Proto Codegen ─────────────────────────────────────────────────────────────

## Regenerate Go stubs from proto/codevaldorg/v1/*.proto.
## Requires: buf, protoc-gen-go, protoc-gen-go-grpc on PATH.
## Install: go install github.com/bufbuild/buf/cmd/buf@latest
##          go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
##          go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
proto:
	buf generate

# ── Tests ─────────────────────────────────────────────────────────────────────

## Run all unit tests (integration tests skip if ArangoDB is unreachable).
test:
	go test -v -race -count=1 ./...

## Run ArangoDB integration tests.
## Loads .env if it exists, otherwise falls back to environment variables.
## Usage: make test-arango
##        ARANGO_ENDPOINTS=http://host:8529 ARANGO_USER=root ARANGO_PASSWORD=pw make test-arango
test-arango:
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a; \
	fi; \
	go test -v -race -count=1 ./internal/server/ ./storage/arangodb/

## Run everything: unit tests + ArangoDB integration tests (loads .env).
test-all:
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a; \
	fi; \
	go test -v -race -count=1 ./...

# ── Quality ───────────────────────────────────────────────────────────────────

vet:
	go vet ./...

lint:
	golangci-lint run ./...

# ── Clean ─────────────────────────────────────────────────────────────────────

clean:
	go clean ./...
	rm -rf bin/
	rm -f coverage.out coverage.html

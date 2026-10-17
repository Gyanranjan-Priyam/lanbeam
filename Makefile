.PHONY: all gen build build-web build-core test test-core test-web lint format dev-web dev-core clean

SHELL := /usr/bin/env bash
GO ?= go
BUN ?= bun

all: gen build

## Code generation
gen: gen-protocol gen-go

gen-protocol:
	$(BUN) run --cwd packages/protocol generate

gen-go:
	cd core && $(GO) run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
		-generate types -package api -o internal/api/types.gen.go ../packages/protocol/openapi.yaml

## Build targets
build: build-share build-core

build-share:
	$(BUN) run --cwd apps/share build

build-web: build-share

build-core:
	cd core && $(GO) build -o ../lanbeam ./cmd/lanbeam

## Testing
test: test-core test-share

test-share:
	$(BUN) run --cwd apps/share test
	$(BUN) run --cwd apps/share check-types

test-web: test-share

## Linting & Formatting
lint: lint-core lint-share

lint-core:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		cd core && golangci-lint run ./...; \
	else \
		cd core && $(GO) vet ./...; \
	fi

lint-share:
	$(BUN) run --cwd apps/share lint

lint-web: lint-share

format:
	cd core && $(GO) fmt ./...
	$(BUN) run prettier --write "**/*.{ts,tsx,js,json,md,yaml,yml}"

## Dev servers
dev-share:
	$(BUN) run --cwd apps/share dev

dev-web: dev-share

dev-core:
	cd core && $(GO) run ./cmd/lanbeam

start-core: dev-core
run: dev-core

clean:
	rm -rf lanbeam core/lanbeam apps/share/out apps/share/.next apps/web/out apps/web/.next

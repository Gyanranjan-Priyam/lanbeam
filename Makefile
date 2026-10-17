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
build: build-web build-core

build-web:
	$(BUN) run --cwd apps/web build

build-core:
	cd core && $(GO) build -o ../lanbeam ./cmd/lanbeam

## Testing
test: test-core test-web

test-core:
	cd core && $(GO) test -v -race ./...

test-web:
	$(BUN) run --cwd apps/web test
	$(BUN) run --cwd apps/web check-types

## Linting & Formatting
lint: lint-core lint-web

lint-core:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		cd core && golangci-lint run ./...; \
	else \
		cd core && $(GO) vet ./...; \
	fi

lint-web:
	$(BUN) run --cwd apps/web lint

format:
	cd core && $(GO) fmt ./...
	$(BUN) run prettier --write "**/*.{ts,tsx,js,json,md,yaml,yml}"

## Dev servers
dev-web:
	$(BUN) run --cwd apps/web dev

dev-core:
	cd core && $(GO) run ./cmd/lanbeam

clean:
	rm -rf lanbeam core/lanbeam apps/web/out apps/web/.next

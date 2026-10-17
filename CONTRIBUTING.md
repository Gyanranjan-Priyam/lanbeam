# Contributing to LanBeam

Thank you for your interest in contributing to LanBeam! LanBeam is a fast, secure, local-network file transfer application and core engine.

## Prerequisites

- **Go**: 1.22+ (tested with 1.27+)
- **Node.js**: LTS (v20+ / v24+)
- **bun**: modern package manager
- **golangci-lint**: for Go linting

## Project Structure

- `core/`: Go transfer engine, TLS listeners, crypto, and CLI.
- `apps/ui/`: React 19 + TypeScript + Tailwind CSS UI (embedded into Go binary).
- `packages/protocol/`: OpenAPI 3.1 specification (`openapi.yaml`) and generated TypeScript client types.
- `docs/`: Architecture specifications, protocol documentation, ADRs, and implementation plans.

## Development Workflow

1. **Install dependencies:**
   ```bash
   bun install
   ```

2. **Generate API schema types:**
   ```bash
   make gen
   # or:
   bun run --cwd packages/protocol generate
   go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate types -package api -o core/internal/api/types.gen.go packages/protocol/openapi.yaml
   ```

3. **Run UI dev server:**
   ```bash
   make dev-ui
   # or: bun run --cwd apps/ui dev
   ```

4. **Run Go core:**
   ```bash
   cd core
   go run ./cmd/lanbeam
   ```

5. **Run tests:**
   ```bash
   make test
   ```

6. **Linting and formatting:**
   ```bash
   make lint
   make format
   ```

## Commit Guidelines

- Ensure `make test` and `make lint` pass before opening a pull request.
- Keep changes focused and self-contained.
- Document any architectural decisions in `docs/adr/`.

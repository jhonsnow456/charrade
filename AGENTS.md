# AGENTS.md

## Project

Charrade — a real-time, browser-based **charades** party game. One player silently acts
out a word while teammates guess it against the clock. Built as a monorepo with a Go
backend (WebSocket realtime) and a React + TypeScript frontend.

## Repository layout

```
apps/
  backend/   Go server — game engine, HTTP API, WebSocket hub
  web/       React + TypeScript + Vite frontend
turbo.json   Turborepo task orchestration
```

This is a Turborepo monorepo. The Go backend participates in Turbo through npm scripts
that wrap `go` commands, so all tasks are run uniformly from the repo root.

## Commands (run from repo root)

| Task           | Command                     | Notes                                   |
| -------------- | --------------------------- | --------------------------------------- |
| Run everything | `pnpm dev`                  | Backend on `:8080`, frontend dev server |
| Backend only   | `pnpm dev --filter=backend` | Go server on `:8080`                    |
| Frontend only  | `pnpm dev --filter=web`     | Vite dev server on `:5173`              |
| Build          | `pnpm build`                | `go build` + `vite build`               |
| Test (TDD)     | `pnpm test`                 | `go test ./...` + `vitest run`          |
| Lint           | `pnpm lint`                 | `go vet ./...` + ESLint                 |
| Format         | `pnpm format`               | Prettier (TS/CSS/JSON; Go uses `gofmt`) |

Single-package equivalents:

```sh
cd apps/backend && go test ./...
cd apps/web && pnpm test -- --run
cd apps/web && pnpm lint
```

## Development workflow (Test-Driven Development)

This repo follows TDD. For any change:

1. Write a failing test first (Go `_test.go` / Vitest `*.test.ts(x)`).
2. Run the focused test and confirm it fails.
3. Implement the minimal code to pass.
4. Run the full test suite for the package before moving on.

## Conventions

### Go (`apps/backend`)

- Module path: `github.com/hey-amanthakur/charrade/apps/backend`
- Stdlib-first. `net/http` + `gorilla/websocket` only; avoid heavyweight frameworks.
- Package layout: `cmd/server` (entrypoint) + `internal/game` (pure game logic, no I/O) +
  `internal/server` (HTTP/WS transport). Keep I/O out of `internal/game`.
- Formatting: `gofmt` / `go vet` — never `go fmt` alternatives.
- Errors: wrap with context (`fmt.Errorf("...: %w", err)`); use `errors.Is`/`errors.As`.
- JSON: field names `camelCase` via struct tags, matching the TS client types.

### TypeScript (`apps/web`)

- Strict mode enabled in `tsconfig.json`. Never loosen `strict`.
- React function components with hooks only (no class components).
- Event/message shapes are shared by contract with the Go server (see
  `src/lib/game.ts` — keep these structs in sync with `internal/game`).
- Import ordering: react, external packages, then local (alphabetical within groups).
- Prettier (root config) + ESLint (flat config in `apps/web/eslint.config.js`).
- CSS: plain CSS files colocated per page/component (mirrors existing style).

### All

- No comments unless they explain non-obvious decisions.
- No secrets in code. Use `VITE_` prefixed env vars, never committed.
- Rebase-friendly commits; keep changes small and focused.

## Tooling notes

- The Go binary and frontend dev server do not auto-restart under `pnpm dev`;
  use `go run` / Vite HMR as appropriate for iteration.
- `packageManager` is pinned to `pnpm@9.12.3` in the root `package.json`.

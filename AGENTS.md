# Repository Guidelines

## Project Structure & Module Organization
- `cmd/api/`: Application entrypoint (`main.go`).
- `internal/app/`: Fx module aggregator.
- `pkg/config/`: Viper-based config loader (YAML + `APP_` env override).
- `pkg/logger/`: Zap logger provider.
- `internal/db/`: GORM Postgres setup and AutoMigrate.
- `internal/models/`: Domain models (e.g., `Profile`).
- `internal/server/`: Gin engine and lifecycle.
- `internal/handlers/`: HTTP route handlers.
- `config/config.yaml`: Sample configuration.

## Build, Test, and Development Commands
- Build: `go build -o bin/api ./cmd/api`
- Run: `make run` (or `bin/api`)
- Tidy deps: `make tidy`
- Format: `make fmt`
- Tests: `go test ./...` (add tests under `internal/...` as needed)

### Debugging (Delve)
- Install/upgrade: `make dlv-install` (uses latest compatible with Go 1.26.0)
- Local debug: `make dlv-debug` (runs `./cmd/api` under dlv)
- Headless attach: `make dlv-headless` (listen `:2345`, API v2)

Requires Go 1.26.0 (`go.mod` sets `go 1.26.0`).

### Required Local Checks (Every Change)
- Proactively run: `go build ./...` immediately after any code change to catch compile errors early, then `go test ./...`.
- If either fails, fix the issues before committing or opening a PR.
- Keep code formatted: `make fmt`; update deps when needed: `make tidy`.
- Build cache rule: run build checks without setting `GOCACHE` (i.e., use plain `go build ./...` with no `GOCACHE` override).

## Coding Style & Naming Conventions
- Formatting: run `make fmt` (gofmt). Keep diffs clean and formatted.
- Package names: lower_snake, short and specific (`server`, `handlers`).
- Files: use `_test.go` for tests; keep one type or concern per file when reasonable.
- Errors: wrap with context; prefer structured logging via `*zap.SugaredLogger`.
- Config keys: lower-case, nested YAML (e.g., `database.dsn`).

## Testing Guidelines
- Framework: standard `testing` with table-driven tests.
- Naming: `TestFunction_Scenario` and helpers in `*_test.go`.
- Run locally: `go test ./...`; aim for deterministic, fast tests.

## Commit & Pull Request Guidelines
- Do not commit proactively: only run `git commit` when the user explicitly asks.
- Commits: concise imperative subject (<=72 chars), body explains why and how.
- Scope: one logical change per commit; include file touches related only to scope.
- PRs: clear description, motivation, and screenshots/logs for visible behavior.
- Link issues (e.g., `Fixes #123`). Note breaking changes in a dedicated section.

## Security & Configuration Tips
- Secrets: never commit keys. Use env vars to override `config/config.yaml` (`APP_` prefix). Example: `APP_DATABASE_DSN=postgres://...`.
- DB: DSN should include `sslmode` as appropriate for your environment.

## Architecture Overview
- HTTP: Gin router.
- IoC: Uber Fx modules wire logger, config, DB, and server.
- Data: PostgreSQL via GORM; migrations run on startup with AutoMigrate.

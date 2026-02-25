# Cashier Backend

## Project Introduction
An out-of-the-box payment collection project that supports multiple payment platforms, offering both one-time and subscription payments.

## Tech Stack (Gin + Fx + Viper + GORM + Zap + Swagger)
- Provides HTTP API using Gin, with Uber Fx for dependency injection and lifecycle management.
- Viper reads `config/config.yaml` and supports environment variable overrides with an `APP_` prefix (`.` mapped to `_`).
- GORM connects to PostgreSQL, executing AutoMigrate automatically on startup.
- Apple IAP Integration: Transaction verification, subscription provisioning, App Store Server Notifications (V2).
- Provides basic statistical interfaces and admin query capabilities; built-in request tracing and access logging.

## Directory Structure
```
cmd/api/main.go                  # Application entry point, starts the Fx app
internal/app/module.go           # Fx module aggregation
internal/app/api/server/http.go  # Gin Engine, route registration, and startup
internal/app/api/handlers/       # HTTP handlers (health, user, admin, webhook)
internal/app/api/middleware/     # Trace/RequestLogger/AccessLog middlewares
internal/app/service/            # Business services (transaction/subscription/statistics/...)
internal/platform/db/postgres.go # GORM Postgres initialization and AutoMigrate
internal/platform/apple/         # Apple IAP/Notification implementations
internal/models/                 # Domain models (Transaction/Subscription/...)
pkg/config/                      # Configuration loading (Viper)
pkg/logger/                      # Logging (Zap)
pkg/response/                    # Unified response structure
pkg/types/                       # Common types (PaymentItem/CommonFilter/...)
docs/                            # Generated Swagger documentation (/swagger)
config/config.yaml               # Example configuration
Makefile                         # Tasks for running, formatting, Swagger, debugging, etc.
```

## Build and Run
- Go Version: 1.26.0 (`go.mod` specifies `go 1.26.0`)
- Build: `go build -o bin/api ./cmd/api`
- Run: `make run` (Equivalent to `go run ./cmd/api`)
- Format: `make fmt`
- Organize dependencies: `make tidy`
- Test: `go test ./...`

## Configuration (YAML + Environment Variable Overrides)
- Reads `config/config.yaml` by default; supports environment variable overrides (prefix `APP_`, e.g., `server.port` -> `APP_SERVER_PORT`).
- Key configurations:
  - `server.host`, `server.port`: Service listening address and port (default `0.0.0.0:8888`).
  - `database.dsn`: PostgreSQL DSN (recommended to set appropriate `sslmode` based on environment).
  - `apple_iap`: Apple IAP keys and switches (production/sandbox).
  - `payment_items`: Items available for sale (corresponding to Provider's Product IDs).

Example (Excerpt):
```yaml
env: dev
server:
  host: 0.0.0.0
  port: 8888
database:
  dsn: postgresql://user:pass@host:5432/db?sslmode=disable
apple_iap:
  key_id: YOUR_KEY_ID
  key_content: |-
    -----BEGIN PRIVATE KEY-----
    ...
    -----END PRIVATE KEY-----
  bundle_id: com.your.app
  issuer: YOUR_ISSUER_ID
  shared_secret: YOUR_SHARED_SECRET
  is_prod: false
payment_items:
  - id: vip_month
    provider_id: apple
    provider_item_id: com.your.app.vip.month
    type: auto_renewable_subscription
    duration_hour: 720 # 3d, used for non-permanent duration products
```

Common Environment Variable Overrides Example:
```
APP_SERVER_PORT=8888
APP_DATABASE_DSN=postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable
APP_APPLE_IAP_IS_PROD=true
```

## API Overview (Actual Routes)
- `GET /healthz`: Health check.
- `GET /swagger/*any`: Swagger UI (Accessed via browser at `/swagger/index.html`).
- Payment Interfaces (`internal/app/api/handlers/payment_v2.go` / `internal/app/api/handlers/payment_webhook.go`)
  - `POST /api/v2/payment/verify_transaction`: Transaction verification (Currently only `provider_id=apple` is supported).
  - `POST /api/v2/payment/webhook/apple`: App Store Server Notification V2 Webhook, Body is the signed JWS text.
- Admin Interfaces (`internal/app/api/handlers/admin.go`, mounted at `/api/v1/admin`):
  - `POST /api/v1/admin/list_user_membership_item`: Paginated/filtered transaction queries (supports `filters/from/size/sort_*`).
  - `POST /api/v1/admin/get_membership_statistic`: Membership/Transaction statistics (Daily GMV, transaction volume, membership volume, retention, etc.).
  - `POST /api/v1/admin/send_free_gift`: Issue a free membership to a user.

Response Wrapper (`pkg/response`):
- Unified structure: `{ code, message, data }`
- Success: `code=0`; Common errors: `40000/50000`.

## Logging and Tracing
- Trace: `TraceMiddleware` injects `traceID`, prioritizing the `X-Request-ID` header, otherwise automatically generating it; writes back the `X-Request-ID` response header.
- Logging: `RequestLoggerMiddleware` injects `*zap.SugaredLogger` with `trace_id` into the context; `AccessLogMiddleware` prints access logs.

## Database and Migrations
- Initialization: `internal/platform/db/postgres.go` connects to PostgreSQL based on `database.dsn`.
- Migrations: AutoMigrates the following models on startup:
  - `Transaction`, `TransactionLog`
  - `UserMembershipActiveItem`
  - `Subscription`, `SubscriptionLog`, `SubscriptionDailySnapshot`
- Note: Please configure the appropriate DSN and permissions based on your runtime environment; SSL is recommended for production.

## Swagger Documentation
- Access: `/swagger/index.html`
- Generate: `make swagger` (requires the `swag` tool).
  - Installation: `go install github.com/swaggo/swag/cmd/swag@latest`
- Documentation meta-information is defined in `cmd/api/main.go`, and endpoint annotations are located in `internal/app/api/handlers`.

## Debugging (Delve)
- Install/Upgrade: `make dlv-install`
- Local debugging: `make dlv-debug`
- Headless: `make dlv-headless` (listens on `:2345`, API v2, IDE attachable)

## Local Development Suggestions
- After each modification, compile first: `go build ./...`, then run tests: `go test ./...`.
- Use `make fmt` to maintain consistent gofmt styling; update dependencies with `make tidy` when necessary.

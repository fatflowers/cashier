# Cashier 后端（Gin + Fx + Viper + GORM + Zap + Swagger）

## 项目简介
- 使用 Gin 提供 HTTP API，Uber Fx 做依赖注入/生命周期管理。
- Viper 读取 `config/config.yaml`，支持 `APP_` 前缀的环境变量覆盖（`.` 映射为 `_`）。
- GORM 连接 PostgreSQL，启动时自动执行 AutoMigrate。
- 集成 Apple IAP：交易核验、订阅发放、App Store Server Notifications（V2）。
- 提供基础统计接口与管理端查询能力；内置请求追踪与访问日志。

## 目录结构
```
cmd/api/main.go                  # 程序入口，启动 Fx 应用
internal/app/module.go           # Fx 模块汇总
internal/app/api/server/http.go  # Gin Engine、路由注册与启动
internal/app/api/handlers/       # HTTP 处理器（health、user、admin、webhook）
internal/app/api/middleware/     # Trace/RequestLogger/AccessLog 中间件
internal/app/service/            # 业务服务（transaction/subscription/statistics/...）
internal/platform/db/postgres.go # GORM Postgres 初始化与 AutoMigrate
internal/platform/apple/         # Apple IAP/通知 相关实现
internal/models/                 # 领域模型（Transaction/Subscription/...）
pkg/config/                      # 配置加载（Viper）
pkg/logger/                      # 日志（Zap）
pkg/response/                    # 统一响应结构
pkg/types/                       # 通用类型（PaymentItem/CommonFilter/...）
docs/                            # 已生成的 Swagger 文档（/swagger）
config/config.yaml               # 示例配置
Makefile                         # 运行、格式化、Swagger、调试等任务
```

## 构建与运行
- Go 版本：1.26.0（`go.mod` 指定 `go 1.26.0`）
- 构建：`go build -o bin/api ./cmd/api`
- 运行：`make run`（等价 `go run ./cmd/api`）
- 格式化：`make fmt`
- 依赖整理：`make tidy`
- 测试：`go test ./...`

## 配置（YAML + 环境变量覆盖）
- 默认读取 `config/config.yaml`；支持环境变量覆盖（前缀 `APP_`，例如 `server.port` -> `APP_SERVER_PORT`）。
- 关键配置项：
  - `server.host`、`server.port`：服务监听地址与端口（默认 `0.0.0.0:8888`）。
  - `database.dsn`：PostgreSQL DSN（建议根据环境设置合适的 `sslmode`）。
  - `apple_iap`：Apple IAP 相关密钥与开关（生产/沙箱）。
  - `payment_items`：可售卖的支付项（与 Provider 商品 ID 对应）。

示例（节选）：
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
    duration_hour: 720 # 3d，用于非永久型时长类商品
```

常用环境变量覆盖示例：
```
APP_SERVER_PORT=8888
APP_DATABASE_DSN=postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable
APP_APPLE_IAP_IS_PROD=true
```

## API 概览（实际路由）
- `GET /healthz`：健康检查。
- `GET /swagger/*any`：Swagger UI（浏览器访问 `/swagger/index.html`）。
- 支付接口（`internal/app/api/handlers/payment_v2.go` / `internal/app/api/handlers/payment_webhook.go`）
  - `POST /api/v2/payment/verify_transaction`：交易核验（本期仅支持 `provider_id=apple`）。
  - `POST /api/v2/payment/webhook/apple`：App Store Server Notification V2 Webhook，Body 为签名的 JWS 文本。
- 管理接口（`internal/app/api/handlers/admin.go`，挂载在 `/api/v1/admin`）：
  - `POST /api/v1/admin/list_user_membership_item`：分页/过滤查询交易（支持 `filters/from/size/sort_*`）。
  - `POST /api/v1/admin/get_membership_statistic`：会员/交易统计（按日 GMV、交易量、会员量、留存等）。
  - `POST /api/v1/admin/send_free_gift`：向用户发放免费会员。

响应包裹（`pkg/response`）：
- 统一结构：`{ code, message, data }`
- 成功：`code=0`；常见错误：`40000/50000`。

## 日志与追踪
- Trace：`TraceMiddleware` 注入 `traceID`，优先读取请求头 `X-Request-ID`，否则自动生成；写回 `X-Request-ID` 响应头。
- 日志：`RequestLoggerMiddleware` 将带 `trace_id` 的 `*zap.SugaredLogger` 注入到上下文；`AccessLogMiddleware` 打印访问日志。

## 数据库与迁移
- 初始化：`internal/platform/db/postgres.go` 基于 `database.dsn` 连接 PostgreSQL。
- 迁移：启动时 AutoMigrate 下列模型：
  - `Transaction`、`TransactionLog`
  - `UserMembershipActiveItem`
  - `Subscription`、`SubscriptionLog`、`SubscriptionDailySnapshot`
- 注意：请根据运行环境配置合适 DSN 与权限；生产环境建议开启 SSL。

## Swagger 文档
- 访问：`/swagger/index.html`
- 生成：`make swagger`（需要安装 `swag` 工具）。
  - 安装：`go install github.com/swaggo/swag/cmd/swag@latest`
- 文档元信息定义于 `cmd/api/main.go`，接口注释位于 `internal/app/api/handlers`。

## 调试（Delve）
- 安装/升级：`make dlv-install`
- 本地调试：`make dlv-debug`
- 头less：`make dlv-headless`（监听 `:2345`，API v2，可供 IDE 附加）

## 本地开发建议
- 每次改动后先编译：`go build ./...`，再跑测试：`go test ./...`。
- 使用 `make fmt` 保持一致的 gofmt 风格；必要时 `make tidy` 更新依赖。

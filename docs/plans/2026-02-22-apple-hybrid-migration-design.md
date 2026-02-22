# Cashier Apple Hybrid Migration 设计文档

日期：2026-02-22

## 1. 背景

当前仓库 `/Users/simon/Documents/GitHub/cashier` 来自早期支付实现；`/Users/simon/work/chat-control` 的支付系统后续已演进。目标是在不引入外部重依赖（Directus/Mongo/奖励系统/OpenIM）的前提下，将 `cashier` 升级为与 `chat-control` 支付核心行为一致的 Apple 支付服务。

本设计是已确认需求的定稿版本，对应后续实施计划：
- 采用 Hybrid（分层迁移）
- 本期仅上线 Apple
- 路由对齐 `chat-control` 风格并移除旧路由
- 数据可重建，不做历史兼容迁移
- `user_id` 继续混合获取策略
- `payment_items` 继续本地配置

## 2. 目标与非目标

### 2.1 目标（本期必须完成）

1. API 契约对齐（Apple 核心）
- 使用 `chat-control` 风格路由（`/api/v2/payment/...`）
- 对齐 verify/webhook 的核心语义与返回结构（至少覆盖降级意图返回）

2. 领域模型对齐（核心表）
- `user_transaction`
- `user_membership`
- `user_membership_active_item`
- `payment_notification_log`

3. 核心业务规则对齐
- Apple verify
- Apple webhook
- 自动续费去重
- 升级识别（upgrade）
- 降级意图识别（downgrade intent）
- 会员有效期切片与投影

### 2.2 非目标（本期不做）

1. `alipay / wechatpay / android / inner` 渠道接入
2. 奖励体系与奖励强耦合响应
3. CMS 化支付商品配置
4. 与 `chat-control` 其他业务域做系统级对齐

## 3. Hybrid 分层迁移架构

### 3.1 契约层（API Contract）

- 对外表现：按 `chat-control` 风格暴露支付路由。
- 对内实现：仍运行在 `cashier` 的 Gin + Fx 结构中。
- 迁移策略：新契约直接替换旧契约（不双轨长期并存）。

### 3.2 应用层（Application Service）

- 增加支付平台服务边界（provider manager dispatch）。
- 保持扩展点，仅启用 Apple manager。
- verify 与 webhook 最终都走统一的“交易入库 -> 会员投影”管线。

### 3.3 领域层（Domain）

- `user_transaction`：交易事实源（含快照、升级关联、续费字段）。
- `user_membership_active_item`：按时间切片后的有效区间。
- `user_membership`：当前会员聚合状态（状态/到期/下次续费）。
- `payment_notification_log`：入站回调与处理结果审计。

### 3.4 基础设施层（Infrastructure）

- 保留：Viper 配置、GORM、PostgreSQL、Fx 生命周期。
- 不新增：外部配置中心、奖励系统、消息推送系统依赖。

## 4. 路由契约

### 4.1 新路由（本期）

1. `POST /api/v2/payment/verify_transaction`
- 入参：`provider_id`, `transaction_id`, `server_verification_data`
- 行为：校验 Apple 交易并完成会员状态投影
- 返回：核心结果；若检测到降级意图，包含降级信息

2. `POST /api/v2/payment/webhook/apple`
- 入参：App Store Server Notification V2 payload
- 行为：解析通知并走统一投影管线

### 4.2 下线路由（本期）

- `/api/v1/user/transaction/verify`
- `/api/v1/user/transaction/list`
- `/api/v1/webhook/apple`

注：下线时机与客户端切换窗口同步。

## 5. 核心数据流

### 5.1 Verify 流程

1. 绑定并校验请求
2. 调用 Apple API 获取/解析交易
3. 从 `AppAccountToken` 解析 `user_id`
4. 映射 `payment_item` 并构建交易快照
5. 自动续费去重检查（同 parent + purchase_at）
6. 升级识别
7. 降级意图识别（`pending_renewal_info + latest_receipt_info`）
8. 开启事务并执行：
- upsert `user_transaction`
- 重算 active slices
- 重建 `user_membership_active_item`
- upsert `user_membership`
9. 返回 verify 结果

### 5.2 Webhook 流程

1. 验签并解析 ASSN V2
2. 提取用户与交易信息
3. 构建交易事实
4. 复用 verify 的同一事务投影管线
5. 记录 received/handled/handle_failed 日志

## 6. 规则对齐细节

### 6.1 自动续费去重

- 使用唯一键 + 业务去重双保险：
- 唯一键：`provider_id + transaction_id`
- 业务去重：`transaction_id != current AND provider_id AND parent_transaction_id AND purchase_at`

### 6.2 升级识别

- 基于 Apple receipt 链中当前 transaction 的后继关系判断升级。
- 升级后的订单应能关联升级前订单（用于区间重排）。

### 6.3 降级意图识别

满足以下条件判定为降级意图：
1. `auto_renew_status == "1"`
2. `auto_renew_product_id != product_id`
3. 能将 `auto_renew_product_id` 映射到本地 payment item
4. 能从对应 receipt 记录解析下个生效时间（expires_date_ms）

### 6.4 有效区间投影

- 输入交易按购买时间排序。
- 非续订与自动续订冲突时，自动续订优先，需切断并顺延后续区间。
- 退款与升级链在区间选择时按 `chat-control` 逻辑处理。
- 最终输出连续有效区间并更新聚合会员状态。

## 7. 错误处理与一致性

### 7.1 错误分层

1. 参数错误：返回业务 BadRequest
2. Apple 校验错误：返回 VerifyFailed
3. 重复交易：返回可识别 Duplicate 语义
4. 降级识别失败：记录 warning（不破坏主流程，除非交易完整性受影响）

### 7.2 一致性边界

以下操作必须同事务提交：
1. 交易 upsert
2. active item 重建
3. membership upsert

日志写入失败不影响主事务提交。

## 8. 测试策略

### 8.1 单元测试

1. 降级识别 helper：
- 命中降级
- auto_renew 关闭
- expires 缺失/非法
- original_transaction_id 不匹配

2. 升级识别与重复交易

3. 区间算法关键场景：
- 自动续费插入打断
- 非续订顺延
- 退款影响
- 升级链影响

### 8.2 集成测试

1. verify 首购 -> 会员生效
2. verify 重复续费 -> duplicate
3. webhook 退款 -> 会员状态更新

### 8.3 变更后必跑

- `go build ./...`
- `go test ./...`

## 9. 风险与缓解

1. 路由切换导致客户端中断
- 缓解：切换窗口内联调 + 路由清单回归

2. 规则细节偏差导致权益计算不一致
- 缓解：关键规则单测先行（TDD）+ 对照样例

3. 模型切换导致迁移噪音
- 缓解：采用可重建策略，优先 clean migration

## 10. 本期交付定义

完成以下即视为本期完成：
1. Apple verify + Apple webhook 可用
2. 新路由契约生效，旧路由移除
3. 核心模型与投影流程对齐
4. duplicate / upgrade / downgrade 三类路径可测可回归
5. 全量编译与测试通过

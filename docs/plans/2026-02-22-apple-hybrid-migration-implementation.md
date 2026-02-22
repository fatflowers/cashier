# Cashier Apple Hybrid Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `cashier` 升级为 Apple-only 的 Hybrid 支付核心，实现与 `chat-control` 对齐的 verify/webhook/会员投影行为，并切换到 `/api/v2/payment` 路由契约。

**Architecture:** 采用分层迁移：先对齐交易校验契约与核心错误语义，再补齐 Apple 降级/升级/去重逻辑，随后切换 API 路由契约并重建会员投影模型。所有状态写入统一走“交易 upsert -> active item 重建 -> membership upsert”事务边界。保持 `cashier` 的 Fx+Gin+GORM 框架，不引入外部系统依赖。

**Tech Stack:** Go 1.26, Gin, Uber Fx, GORM, PostgreSQL, go-iap (`github.com/awa/go-iap/appstore`), testify。

---

执行提示：实现阶段请使用 `@executing-plans`，严格按任务顺序执行，保持 DRY/YAGNI/TDD，小步提交。

### Task 1: 扩展 Verify 契约（结果对象 + 降级判断）

**Files:**
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/manager.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/manager_verify_result_test.go`

**Step 1: Write the failing test**

```go
package transaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerifyTransactionResult_IsDowngrade(t *testing.T) {
	next := time.Unix(1735689600, 0)
	res := &VerifyTransactionResult{
		DowngradeToVipID:         "vip_low",
		DowngradeNextAutoRenewAt: &next,
	}
	require.True(t, res.IsDowngrade())
}

func TestVerifyTransactionResult_IsDowngrade_FalseWhenMissingFields(t *testing.T) {
	res := &VerifyTransactionResult{DowngradeToVipID: ""}
	require.False(t, res.IsDowngrade())
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/service/transaction -run TestVerifyTransactionResult_IsDowngrade -v`
Expected: FAIL with `undefined: VerifyTransactionResult` or `res.IsDowngrade undefined`.

**Step 3: Write minimal implementation**

```go
// manager.go
package transaction

import (
	"context"
	"time"

	models "github.com/fatflowers/cashier/internal/models"
)

type VerifyTransactionResult struct {
	DowngradeToVipID         string              `json:"downgrade_to_vip_id,omitempty"`
	DowngradeNextAutoRenewAt *time.Time          `json:"downgrade_next_auto_renew_at,omitempty"`
	IsUpgrade                bool                `json:"is_upgrade,omitempty"`
	UserTransaction          *models.Transaction `json:"user_transaction,omitempty"`
}

func (r *VerifyTransactionResult) IsDowngrade() bool {
	return r != nil && r.DowngradeToVipID != "" && r.DowngradeNextAutoRenewAt != nil && !r.DowngradeNextAutoRenewAt.IsZero()
}

// interface change
// VerifyTransaction(ctx context.Context, req *TransactionVerifyRequest) (*VerifyTransactionResult, error)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/service/transaction -run TestVerifyTransactionResult_IsDowngrade -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/manager.go /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/manager_verify_result_test.go
git commit -m "feat: add verify result contract with downgrade helper"
```

### Task 2: 增加可识别的重复交易错误语义

**Files:**
- Create: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/errors.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/errors_test.go`

**Step 1: Write the failing test**

```go
package transaction

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrVerifyTransactionDuplicate_IsWrapFriendly(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", ErrVerifyTransactionDuplicate)
	require.True(t, errors.Is(err, ErrVerifyTransactionDuplicate))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/service/transaction -run TestErrVerifyTransactionDuplicate_IsWrapFriendly -v`
Expected: FAIL with `undefined: ErrVerifyTransactionDuplicate`.

**Step 3: Write minimal implementation**

```go
package transaction

import "errors"

var ErrVerifyTransactionDuplicate = errors.New("verify transaction duplicate")
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/service/transaction -run TestErrVerifyTransactionDuplicate_IsWrapFriendly -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/errors.go /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/errors_test.go
git commit -m "feat: add duplicate transaction sentinel error"
```

### Task 3: 将 VerifyData 收据类型对齐为 `appstore.IAPResponse`

**Files:**
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/manager.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/platform/apple/apple_iap/iap.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/verified_data_type_test.go`

**Step 1: Write the failing test**

```go
package transaction

import (
	"testing"

	"github.com/awa/go-iap/appstore"
	"github.com/stretchr/testify/require"
)

func TestVerifiedData_UsesAppstoreIAPResponse(t *testing.T) {
	got := &VerifiedData{AppleReceipt: &appstore.IAPResponse{}}
	require.NotNil(t, got.AppleReceipt)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/service/transaction -run TestVerifiedData_UsesAppstoreIAPResponse -v`
Expected: FAIL with type mismatch for `AppleReceipt`.

**Step 3: Write minimal implementation**

```go
// manager.go
import "github.com/awa/go-iap/appstore"

type VerifiedData struct {
	AppleReceipt *appstore.IAPResponse
}

// iap.go
func VerifyServerVerificationData(ctx context.Context, receiptData string, opts *GetAppleIAPClientOptions) (*appstore.IAPResponse, error) {
	client := appstore.New()
	if opts.Sandbox {
		client.ProductionURL = client.SandboxURL
	}
	var result appstore.IAPResponse
	err := client.Verify(ctx, appstore.IAPRequest{
		ReceiptData:            receiptData,
		Password:               opts.SharedSecret,
		ExcludeOldTransactions: true,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to verify receipt: %w", err)
	}
	return &result, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/service/transaction -run TestVerifiedData_UsesAppstoreIAPResponse -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/manager.go /Users/simon/Documents/GitHub/cashier/internal/platform/apple/apple_iap/iap.go /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/verified_data_type_test.go
git commit -m "refactor: align verified receipt type with appstore iap response"
```

### Task 4: 增加 Apple 降级识别 Helper（TDD）

**Files:**
- Create: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple_downgrade.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple_downgrade_test.go`

**Step 1: Write the failing test**

```go
package transaction

import (
	"context"
	"testing"
	"time"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
	types "github.com/fatflowers/cashier/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestDetectAppleDowngrade_Success(t *testing.T) {
	ctx := context.Background()

	restore := applePaymentItemLookup
	defer func() { applePaymentItemLookup = restore }()
	applePaymentItemLookup = func(_ context.Context, _ types.PaymentProvider, providerItemID string) (*types.PaymentItem, error) {
		require.Equal(t, "vip.low.month", providerItemID)
		return &types.PaymentItem{ID: "vip_low"}, nil
	}

	parseResult := &VerifiedData{AppleReceipt: &appstore.IAPResponse{
		PendingRenewalInfo: []appstore.PendingRenewalInfo{{
			OriginalTransactionID:        "orig-1",
			ProductID:                    "vip.high.month",
			SubscriptionAutoRenewProductID: "vip.low.month",
			SubscriptionAutoRenewStatus:  "1",
		}},
		LatestReceiptInfo: []appstore.InApp{{
			OriginalTransactionID: "orig-1",
			ProductID:             "vip.high.month",
			ExpiresDate:           appstore.ExpiresDate{ExpiresDateMS: "1770724800000"},
		}},
	}}

	vipID, nextAt, ok, err := detectAppleDowngrade(ctx, parseResult, &api.JWSTransaction{OriginalTransactionId: "orig-1"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "vip_low", vipID)
	require.Equal(t, time.UnixMilli(1770724800000), *nextAt)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/service/transaction -run TestDetectAppleDowngrade_Success -v`
Expected: FAIL with `undefined: detectAppleDowngrade`.

**Step 3: Write minimal implementation**

```go
package transaction

import (
	"context"
	"strconv"
	"time"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
	types "github.com/fatflowers/cashier/pkg/types"
)

var applePaymentItemLookup = func(ctx context.Context, provider types.PaymentProvider, providerItemID string) (*types.PaymentItem, error) {
	return nil, nil
}

func detectAppleDowngrade(ctx context.Context, parseResult *VerifiedData, txInfo *api.JWSTransaction) (string, *time.Time, bool, error) {
	if parseResult == nil || parseResult.AppleReceipt == nil || txInfo == nil {
		return "", nil, false, nil
	}

	var pending *appstore.PendingRenewalInfo
	for i := range parseResult.AppleReceipt.PendingRenewalInfo {
		v := &parseResult.AppleReceipt.PendingRenewalInfo[i]
		if v.OriginalTransactionID == txInfo.OriginalTransactionId {
			pending = v
			break
		}
	}
	if pending == nil || pending.SubscriptionAutoRenewStatus != "1" || pending.ProductID == pending.SubscriptionAutoRenewProductID {
		return "", nil, false, nil
	}

	var latestMS int64
	for _, info := range parseResult.AppleReceipt.LatestReceiptInfo {
		if info.OriginalTransactionID != pending.OriginalTransactionID || info.ProductID != pending.ProductID || info.ExpiresDateMS == "" {
			continue
		}
		ms, err := strconv.ParseInt(info.ExpiresDateMS, 10, 64)
		if err == nil && ms > latestMS {
			latestMS = ms
		}
	}
	if latestMS == 0 {
		return "", nil, false, nil
	}

	item, err := applePaymentItemLookup(ctx, types.PaymentProviderApple, pending.SubscriptionAutoRenewProductID)
	if err != nil || item == nil || item.ID == "" {
		return "", nil, false, err
	}

	next := time.UnixMilli(latestMS)
	return item.ID, &next, true, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/service/transaction -run TestDetectAppleDowngrade_Success -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple_downgrade.go /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple_downgrade_test.go
git commit -m "feat: add apple downgrade detection helper"
```

### Task 5: Apple Verify 返回结构化结果并接入 duplicate/downgrade

**Files:**
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/service.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple_verify_duplicate_test.go`

**Step 1: Write the failing test**

```go
package transaction

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDuplicateError_IsMapped(t *testing.T) {
	err := mapDuplicateErr("duplicate transaction already exists: tx-1")
	require.True(t, errors.Is(err, ErrVerifyTransactionDuplicate))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/service/transaction -run TestDuplicateError_IsMapped -v`
Expected: FAIL with `undefined: mapDuplicateErr`.

**Step 3: Write minimal implementation**

```go
// apple.go
func mapDuplicateErr(msg string) error {
	if strings.Contains(msg, "duplicate transaction already exists") {
		return fmt.Errorf("%w: %s", ErrVerifyTransactionDuplicate, msg)
	}
	return errors.New(msg)
}

func (a *AppleTransactionManager) VerifyTransaction(ctx context.Context, req *TransactionVerifyRequest) (*VerifyTransactionResult, error) {
	result := &VerifyTransactionResult{}
	// ... existing logic
	if exists {
		return nil, fmt.Errorf("%w: %s", ErrVerifyTransactionDuplicate, txInfo.TransactionID)
	}
	// when downgrade detected:
	result.DowngradeToVipID = vipID
	result.DowngradeNextAutoRenewAt = downgradeAt
	result.UserTransaction = mappedItem
	return result, nil
}

// service.go
func (s *Service) VerifyTransaction(ctx context.Context, req *TransactionVerifyRequest) (*VerifyTransactionResult, error) {
	switch req.ProviderID {
	case string(types.PaymentProviderApple):
		return s.appleTransactionManager.VerifyTransaction(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.ProviderID)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/service/transaction -run TestDuplicateError_IsMapped -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple.go /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/service.go /Users/simon/Documents/GitHub/cashier/internal/app/service/transaction/apple_verify_duplicate_test.go
git commit -m "feat: return structured verify result with duplicate semantics"
```

### Task 6: 新增 v2 Verify Handler 并返回降级信息

**Files:**
- Create: `/Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_v2.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_v2_test.go`

**Step 1: Write the failing test**

```go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/fatflowers/cashier/internal/app/service/transaction"
)

type stubTxMgr struct{}

func (s *stubTxMgr) VerifyTransaction(_ context.Context, _ *transaction.TransactionVerifyRequest) (*transaction.VerifyTransactionResult, error) {
	next := time.Unix(1735689600, 0)
	return &transaction.VerifyTransactionResult{
		DowngradeToVipID:         "vip_low",
		DowngradeNextAutoRenewAt: &next,
	}, nil
}

// implement other interface methods with panic("not used")

func TestApiVerifyTransactionV2_ReturnsDowngradeInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v2/payment/verify_transaction", ApiVerifyTransactionV2(&stubTxMgr{}))

	body, _ := json.Marshal(map[string]any{"provider_id": "apple", "transaction_id": "tx-1", "server_verification_data": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/payment/verify_transaction", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "down_grade_auto_renew_info")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/api/handlers -run TestApiVerifyTransactionV2_ReturnsDowngradeInfo -v`
Expected: FAIL with `undefined: ApiVerifyTransactionV2`.

**Step 3: Write minimal implementation**

```go
package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/fatflowers/cashier/internal/app/service/transaction"
	"github.com/fatflowers/cashier/pkg/response"
	"github.com/gin-gonic/gin"
)

type downGradeAutoRenewInfo struct {
	VipID           string    `json:"vip_id"`
	NextAutoRenewAt time.Time `json:"next_auto_renew_at"`
}

type verifyTransactionV2Resp struct {
	DownGradeAutoRenewInfo *downGradeAutoRenewInfo `json:"down_grade_auto_renew_info,omitempty"`
}

func ApiVerifyTransactionV2(mgr transaction.TransactionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req transaction.TransactionVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
			return
		}

		res, err := mgr.VerifyTransaction(c.Request.Context(), &req)
		if err != nil {
			if errors.Is(err, transaction.ErrVerifyTransactionDuplicate) {
				c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
				return
			}
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}

		out := verifyTransactionV2Resp{}
		if res != nil && res.IsDowngrade() {
			out.DownGradeAutoRenewInfo = &downGradeAutoRenewInfo{VipID: res.DowngradeToVipID, NextAutoRenewAt: *res.DowngradeNextAutoRenewAt}
		}
		c.JSON(http.StatusOK, response.OKT(out))
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/api/handlers -run TestApiVerifyTransactionV2_ReturnsDowngradeInfo -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_v2.go /Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_v2_test.go
git commit -m "feat: add payment verify v2 handler with downgrade response"
```

### Task 7: 路由切换到 `/api/v2/payment` 并移除旧支付路由

**Files:**
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/api/server/http.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_webhook.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/user.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_routes_test.go`

**Step 1: Write the failing test**

```go
package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterPaymentV2Routes_RegistersEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v2/payment")
	RegisterPaymentV2Routes(g, nil, nil)

	routes := r.Routes()
	contains := func(target string) bool {
		for _, rt := range routes {
			if rt.Method+" "+rt.Path == target {
				return true
			}
		}
		return false
	}

	require.True(t, contains("POST /api/v2/payment/verify_transaction"))
	require.True(t, contains("POST /api/v2/payment/webhook/apple"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/api/handlers -run TestRegisterPaymentV2Routes_RegistersEndpoints -v`
Expected: FAIL with `undefined: RegisterPaymentV2Routes`.

**Step 3: Write minimal implementation**

```go
// in handlers/payment_v2.go
func RegisterPaymentV2Routes(r gin.IRouter, txMgr transaction.TransactionManager, notif *nh.NotificationHandler) {
	r.POST("/verify_transaction", ApiVerifyTransactionV2(txMgr))
	r.POST("/webhook/apple", ApiAppleWebhook(notif))
}

// in server/http.go
paymentV2 := r.Group("/api/v2/payment")
paymentV2.Use(mw.RequestLoggerMiddleware(log), mw.AccessLogMiddleware())
handlers.RegisterPaymentV2Routes(paymentV2, txMgr, notifHandler)

// remove or stop registering legacy:
// handlers.RegisterPaymentWebhookRoutes(apiV1.Group("/webhook"), notifHandler)
// handlers.RegisterTransactionRoutes(apiV1.Group("/user"), txMgr)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/api/handlers -run TestRegisterPaymentV2Routes_RegistersEndpoints -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/api/server/http.go /Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_webhook.go /Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/user.go /Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_routes_test.go /Users/simon/Documents/GitHub/cashier/internal/app/api/handlers/payment_v2.go
git commit -m "feat: cut over payment routes to api v2 contract"
```

### Task 8: 引入 `user_membership_active_item` 模型并纳入 AutoMigrate

**Files:**
- Create: `/Users/simon/Documents/GitHub/cashier/internal/models/model_user_membership_active_item.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/platform/db/postgres.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/models/model_user_membership_active_item_test.go`

**Step 1: Write the failing test**

```go
package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserMembershipActiveItem_TableName(t *testing.T) {
	var m UserMembershipActiveItem
	require.Equal(t, "user_membership_active_item", m.TableName())
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/models -run TestUserMembershipActiveItem_TableName -v`
Expected: FAIL with `undefined: UserMembershipActiveItem`.

**Step 3: Write minimal implementation**

```go
package models

import "time"

type UserMembershipActiveItem struct {
	ID                       string     `gorm:"column:id;type:uuid;primaryKey"`
	UserTransactionID        string     `gorm:"column:user_transaction_id;type:uuid;not null;index"`
	PaymentItemID            string     `gorm:"column:payment_item_id;type:varchar(64);not null"`
	UserID                   string     `gorm:"column:user_id;type:varchar(64);not null;index:idx_user_active_time,priority:1"`
	RemainingDurationSeconds int64      `gorm:"column:remaining_duration_seconds;type:bigint;not null"`
	ActivatedAt              time.Time  `gorm:"column:activated_at;not null;index:idx_user_active_time,priority:2"`
	ExpireAt                 time.Time  `gorm:"column:expire_at;not null;index:idx_user_active_time,priority:3"`
	NextAutoRenewAt          *time.Time `gorm:"column:next_auto_renew_at"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (UserMembershipActiveItem) TableName() string {
	return "user_membership_active_item"
}

// postgres.go AutoMigrate add: &models.UserMembershipActiveItem{}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/models -run TestUserMembershipActiveItem_TableName -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/models/model_user_membership_active_item.go /Users/simon/Documents/GitHub/cashier/internal/models/model_user_membership_active_item_test.go /Users/simon/Documents/GitHub/cashier/internal/platform/db/postgres.go
git commit -m "feat: add user membership active item model and migration"
```

### Task 9: 会员投影流程重建 active items（事务内）

**Files:**
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/subscription/subscription.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/subscription/subscription_item.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/service/subscription/membership_item_test.go`

**Step 1: Write the failing test**

```go
func TestGetAllActiveUserMembershipItems_AutoRenewOverlap_UsesActivatedAtForRemainingDuration(t *testing.T) {
	now := time.Now()
	oneMonthHours := int64(30 * 24)

	cfg := &config.Config{PaymentItems: []*types.PaymentItem{
		{ID: "p1", Type: types.PaymentItemTypeNonRenewableSubscription, DurationHour: &oneMonthHours},
		{ID: "p2", Type: types.PaymentItemTypeAutoRenewableSubscription},
	}}
	svc := NewService(cfg, nil, zap.NewNop().Sugar())

	txs := []*models.Transaction{
		{ID: "1", PaymentItemID: "p1", PurchaseAt: now},
		{ID: "2", PaymentItemID: "p2", PurchaseAt: now.Add(15 * 24 * time.Hour), AutoRenewExpireAt: &[]time.Time{now.Add(45 * 24 * time.Hour)}[0]},
	}

	items, err := svc.getAllActiveUserSubscriptionItems(context.Background(), txs, now.Add(20*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, int64((15*24*time.Hour).Seconds()), items[1].RemainingDurationSeconds)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/service/subscription -run TestGetAllActiveUserMembershipItems_AutoRenewOverlap_UsesActivatedAtForRemainingDuration -v`
Expected: FAIL with remaining duration mismatch.

**Step 3: Write minimal implementation**

```go
// subscription_item.go in processAutoRenewableSubscription overlapping branch
remainingDuration := result[index].ExpireAt.Sub(item.ActivatedAt)
result[index].RemainingDurationSeconds = int64(remainingDuration.Seconds())
result[index].ActivatedAt = item.ExpireAt
result[index].ExpireAt = result[index].ActivatedAt.Add(remainingDuration)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/service/subscription -run TestGetAllActiveUserMembershipItems_AutoRenewOverlap_UsesActivatedAtForRemainingDuration -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/service/subscription/subscription.go /Users/simon/Documents/GitHub/cashier/internal/app/service/subscription/subscription_item.go /Users/simon/Documents/GitHub/cashier/internal/app/service/subscription/membership_item_test.go
git commit -m "fix: align membership overlap remaining duration calculation"
```

### Task 10: Webhook 对齐与回归验证

**Files:**
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/notification_handler/handler.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/internal/app/service/notification_handler/apple_parser.go`
- Test: `/Users/simon/Documents/GitHub/cashier/internal/app/service/notification_handler/apple_parser_test.go`
- Modify: `/Users/simon/Documents/GitHub/cashier/README.md`

**Step 1: Write the failing test**

```go
package notification_handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppleNotificationParser_GetUserID_EmptyToken(t *testing.T) {
	p := &AppleNotificationParser{Notification: &apple_notification.AppStoreServerNotification{TransactionInfo: &apple_notification.JWSTransactionDecodedPayload{AppAccountToken: ""}}}
	_, err := p.GetUserID(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "app account token is empty")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/service/notification_handler -run TestAppleNotificationParser_GetUserID_EmptyToken -v`
Expected: FAIL if parser behavior drifts.

**Step 3: Write minimal implementation**

```go
// keep parser strict and ensure handler logs lifecycle statuses
// handler.go should always write:
// received -> handled on success, received -> handle_failed on any error
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app/service/notification_handler -run TestAppleNotificationParser_GetUserID_EmptyToken -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/internal/app/service/notification_handler/handler.go /Users/simon/Documents/GitHub/cashier/internal/app/service/notification_handler/apple_parser.go /Users/simon/Documents/GitHub/cashier/internal/app/service/notification_handler/apple_parser_test.go /Users/simon/Documents/GitHub/cashier/README.md
git commit -m "chore: align apple webhook parser and lifecycle logging"
```

### Task 11: 全量检查与文档收尾

**Files:**
- Modify: `/Users/simon/Documents/GitHub/cashier/docs/swagger.yaml`
- Modify: `/Users/simon/Documents/GitHub/cashier/docs/swagger.json`
- Modify: `/Users/simon/Documents/GitHub/cashier/docs/docs.go`

**Step 1: Write the failing check**

```bash
# intentional pre-check before regeneration
go test ./...
```

**Step 2: Run check to verify current drift**

Run: `go test ./...`
Expected: If Swagger annotations changed but docs not regenerated, CI/doc consistency checks should reveal drift.

**Step 3: Write minimal implementation**

```bash
make swagger
make fmt
```

**Step 4: Run checks to verify it passes**

Run: `go build ./...`
Expected: PASS.

Run: `go test ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/simon/Documents/GitHub/cashier/docs/swagger.yaml /Users/simon/Documents/GitHub/cashier/docs/swagger.json /Users/simon/Documents/GitHub/cashier/docs/docs.go
git commit -m "docs: regenerate swagger after payment v2 contract cutover"
```

## Final Validation Checklist

1. `POST /api/v2/payment/verify_transaction` 可返回降级信息。
2. `POST /api/v2/payment/webhook/apple` 可正常入库并更新会员状态。
3. duplicate/upgrade/downgrade 路径均有测试覆盖。
4. 旧支付路由已移除。
5. `go build ./...` 与 `go test ./...` 全通过。

package handlers

import (
	"net/http"
	"github.com/fatflowers/cashier/internal/app/service/statistics"
	subsvc "github.com/fatflowers/cashier/internal/app/service/subscription"
	"github.com/fatflowers/cashier/internal/app/service/transaction"
	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/config"
	"github.com/fatflowers/cashier/pkg/response"
	"github.com/fatflowers/cashier/pkg/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

type ListTransactionRequest struct {
	Filters   []*types.CommonFilter `json:"filters"`
	From      int                   `json:"from"`
	Size      int                   `json:"size"`
	SortBy    string                `json:"sort_by"`
	SortOrder string                `json:"sort_order"`
}

type TransactionItem struct {
	ID                  string                `json:"id"`
	TransactionID       string                `json:"transaction_id"`
	UserID              string                `json:"user_id"`
	Currency            string                `json:"currency"`
	Price               int64                 `json:"price"`
	ProviderID          types.PaymentProvider `json:"provider_id"`
	IsFirstPurchase     bool                  `json:"is_first_purchase"`
	PurchaseAt          time.Time             `json:"purchase_at"`
	RefundAt            *time.Time            `json:"refund_at"`
	NextAutoRenewAt     *time.Time            `json:"next_auto_renew_at"`
	AutoRenewExpireAt   *time.Time            `json:"auto_renew_expire_at"`
	ParentTransactionID *string               `json:"parent_transaction_id"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
	PaymentItemID       string                `json:"payment_item_id"`
	PaymentItemType     types.PaymentItemType `json:"payment_item_type"`
	ProviderItemID      string                `json:"provider_item_id"`
	DurationMinutes     int64                 `json:"membership_duration_minutes"`
}

// filtersWhere wraps a list of filters to a single clause.Expression
type filtersWhere struct{ filters []*types.CommonFilter }

func (w filtersWhere) Build(builder clause.Builder) {
	if len(w.filters) == 0 {
		builder.WriteString("1=1")
		return
	}
	for i, f := range w.filters {
		if i > 0 {
			builder.WriteString(" AND ")
		}
		f.Build(builder)
	}
}

func toTransactionItem(cfg *config.Config, m *models.Transaction) *TransactionItem {
	var t types.PaymentItemType
	var providerItemID string
	var durMinutes int64
	if snap := m.GetPaymentItemSnapshot(); snap != nil {
		t = snap.Type
		providerItemID = snap.ProviderItemID
		if snap.DurationHour != nil {
			durMinutes = int64(*snap.DurationHour * 60)
		}
	} else if cfg != nil {
		if pi := cfg.GetPaymentItemByID(m.PaymentItemID); pi != nil {
			t = pi.Type
			providerItemID = pi.ProviderItemID
			if pi.DurationHour != nil {
				durMinutes = int64(*pi.DurationHour * 60)
			}
		}
	}

	return &TransactionItem{
		ID:            m.ID,
		TransactionID: m.TransactionID,
		UserID:        m.UserID,
		Currency:      m.Currency,
		Price:         m.Price,
		ProviderID:    m.ProviderID,
		IsFirstPurchase: func() bool {
			if e := m.Extra.Data(); e != nil {
				return e.IsFirstPurchase
			}
			return false
		}(),
		PurchaseAt:          m.PurchaseAt,
		RefundAt:            m.RefundAt,
		NextAutoRenewAt:     m.NextAutoRenewAt,
		AutoRenewExpireAt:   m.AutoRenewExpireAt,
		ParentTransactionID: m.ParentTransactionID,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
		PaymentItemID:       m.PaymentItemID,
		PaymentItemType:     t,
		ProviderItemID:      providerItemID,
		DurationMinutes:     durMinutes,
	}
}

type ListMembershipTransactionsResponse struct {
	Items []*TransactionItem `json:"items"`
	Total int64              `json:"total"`
}

// @Summary      List Membership Transactions (Admin)
// @Description  Retrieves a paginated and filterable list of all membership transactions.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request body ListTransactionRequest true "List transaction request with filters, pagination, and sorting"
// @Success      200  {object}  handlers.RespListMembershipTransactions
// @Router       /api/v1/admin/list_user_membership_item [post]
func ApiListMembershipTransactions(mgr transaction.TransactionManager, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ListTransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
			return
		}
		scanReq := &transaction.ScanTransactionsRequest{Filters: req.Filters, From: req.From, Size: req.Size, SortBy: req.SortBy, SortOrder: req.SortOrder}
		res, err := mgr.ScanTransactions(c.Request.Context(), scanReq)
		if err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}
		items := lo.Map(res.Items, func(it *models.Transaction, _ int) *TransactionItem { return toTransactionItem(cfg, it) })
		c.JSON(http.StatusOK, response.OKT(&ListMembershipTransactionsResponse{Items: items, Total: res.Total}))
	}
}

// @Summary      Get Membership Statistics (Admin)
// @Description  Retrieves daily membership statistics.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request body statistics.MembershipStatisticRequest true "Statistic request parameters"
// @Success      200  {object}  handlers.RespMembershipStatistic
// @Router       /api/v1/admin/get_membership_statistic [post]
// ApiGetMembershipStatistic handles POST /v1/admin/get_membership_statistic
func ApiGetMembershipStatistic(svc *statistics.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req statistics.MembershipStatisticRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
			return
		}
		res, err := svc.GetDailyMembershipStatistic(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}
		c.JSON(http.StatusOK, response.OKT(res))
	}
}

// @Summary      Send Free Gift (Admin)
// @Description  Grants a free membership item to a user.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request body transaction.SendFreeGiftRequest true "Send free gift request"
// @Success      200  {object}  handlers.RespOK
// @Router       /api/v1/admin/send_free_gift [post]
// ApiSendFreeGift handles POST /api/v1/send_free_gift
func ApiSendFreeGift(sub *subsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID        string `json:"user_id"`
			PaymentItemID string `json:"payment_item_id"`
			OperatorId    string `json:"operator_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
			return
		}
		if req.UserID == "" || req.PaymentItemID == "" || req.OperatorId == "" {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, "missing user_id or payment_item_id or operator_id"))
			return
		}
		if err := sub.SendFreeGift(c.Request.Context(), req.UserID, req.PaymentItemID, req.OperatorId); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}
		c.JSON(http.StatusOK, response.OKT[any](nil))
	}
}

func RegisterAdminPaymentRoutes(r gin.IRouter, mgr transaction.TransactionManager, cfg *config.Config, stats *statistics.Service, sub *subsvc.Service) {
	r.POST("/list_user_membership_item", ApiListMembershipTransactions(mgr, cfg))
	r.POST("/get_membership_statistic", ApiGetMembershipStatistic(stats))
	r.POST("/send_free_gift", ApiSendFreeGift(sub))
}

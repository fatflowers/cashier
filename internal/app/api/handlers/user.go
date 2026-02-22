package handlers

import (
	"github.com/fatflowers/cashier/internal/app/service/transaction"
	"github.com/fatflowers/cashier/pkg/response"
	types "github.com/fatflowers/cashier/pkg/types"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @Summary      Verify Transaction
// @Description  Verifies a payment transaction with the provider and grants entitlement.
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        request body transaction.TransactionVerifyRequest true "Transaction verification request"
// @Success      200  {object}  handlers.RespOK "Successful verification"
// @Router       /api/v1/user/transaction/verify [post]
// ApiVerifyTransaction handles POST /api/v1/verify_transaction
func ApiVerifyTransaction(mgr transaction.TransactionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req transaction.TransactionVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, err.Error()))
			return
		}
		if err := mgr.VerifyTransaction(c.Request.Context(), &req); err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}
		c.JSON(http.StatusOK, response.OKT[any](nil))
	}
}

// @Summary      List User Transactions
// @Description  Retrieves a paginated list of transactions for a specific user.
// @Tags         User
// @Produce      json
// @Param        user_id    query     string  true  "User ID"
// @Param        from       query     int     false "Pagination offset" default(0)
// @Param        size       query     int     false "Pagination limit" default(100)
// @Param        sort_by    query     string  false "Sort by field" default(purchase_at)
// @Param        sort_order query     string  false "Sort order (asc/desc)" default(desc) Enums(asc, desc)
// @Success      200        {object}  handlers.RespUserListTransactions
// @Router       /api/v1/user/transaction/list [get]
// ApiTransactionList handles GET /api/v1/transaction/list
func ApiTransactionList(mgr transaction.TransactionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, "missing user_id"))
			return
		}
		// Build filter to match user_id and sort by purchase_at desc
		// Read pagination from query params
		from := 0
		if v := c.Query("from"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				from = n
			}
		}
		size := 100
		if v := c.Query("size"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				size = n
			} else {
				c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeBadRequest, "invalid size"))
				return
			}
		}
		// Sorting from query with defaults
		sortBy := c.Query("sort_by")
		if sortBy == "" {
			sortBy = "purchase_at"
		}
		sortOrder := c.Query("sort_order")
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "desc"
		}

		req := &transaction.ScanTransactionsRequest{
			Filters:   []*types.CommonFilter{{Field: "user_id", Operator: types.CommonFilterOperatorEq, Values: []any{userID}}},
			From:      from,
			Size:      size,
			SortBy:    sortBy,
			SortOrder: sortOrder,
		}
		res, err := mgr.ScanTransactions(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusOK, response.ErrorT[any](response.APIResponseCodeError, err.Error()))
			return
		}
		c.JSON(http.StatusOK, response.OKT(res.Items))
	}
}

func RegisterTransactionRoutes(r gin.IRouter, mgr transaction.TransactionManager) {
	r.POST("/transaction/verify", ApiVerifyTransaction(mgr))
	r.GET("/transaction/list", ApiTransactionList(mgr))
}

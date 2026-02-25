package handlers

import (
	"github.com/fatflowers/cashier/internal/app/service/transaction"
	"github.com/fatflowers/cashier/pkg/response"
	types "github.com/fatflowers/cashier/pkg/types"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ApiTransactionList is a legacy helper kept for internal reuse.
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
	r.GET("/transaction/list", ApiTransactionList(mgr))
}

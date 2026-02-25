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

package handlers

import (
	"net/http"
	"github.com/fatflowers/cashier/pkg/response"

	"github.com/gin-gonic/gin"
)

// @Summary      Health check
// @Description  Returns service status
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /healthz [get]
func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, response.OKT(map[string]string{"status": "ok"}))
}

func RegisterHealthRoutes(r gin.IRouter) {
	r.GET("/healthz", Healthz)
}

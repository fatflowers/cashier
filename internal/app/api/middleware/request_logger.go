package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLoggerMiddleware attaches a request-scoped logger enriched with
// trace_id and user_id (if present) to gin.Context and request context.
func RequestLoggerMiddleware(base *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID, _ := c.Get("traceID")

		reqLogger := base.With("trace_id", traceID)
		c.Set("logger", reqLogger)

		// also attach to std context
		ctx := context.WithValue(c.Request.Context(), "logger", reqLogger)
		c.Request = c.Request.WithContext(ctx)

		// mirror trace id to response header when available
		if s, ok := traceID.(string); ok && s != "" {
			c.Writer.Header().Set("X-Request-ID", s)
		}

		c.Next()
	}
}

package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLogMiddleware logs HTTP access using the request-scoped logger
// previously attached by RequestLoggerMiddleware.
func AccessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		// pick request logger if present
		if l, ok := c.Get("logger"); ok {
			if log, ok := l.(*zap.SugaredLogger); ok && log != nil {
				log.Infow("http_access",
					"method", c.Request.Method,
					"path", c.FullPath(),
					"status", c.Writer.Status(),
					"latency_ms", latency.Milliseconds(),
					"client_ip", c.ClientIP(),
				)
				return
			}
		}
	}
}

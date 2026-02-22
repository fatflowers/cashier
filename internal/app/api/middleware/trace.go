package middleware

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceMiddleware adds a trace ID to the request context.
// It reads X-Request-ID if provided by the client; otherwise generates a UUID.
// The trace ID is stored in both gin.Context (key: "traceID") and the request's context.Context.
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Attach to gin context and request context
		c.Set("traceID", traceID)
		ctx := context.WithValue(c.Request.Context(), "traceID", traceID)
		c.Request = c.Request.WithContext(ctx)

		log.Printf("Request with TraceID: %s", traceID)
		c.Next()
	}
}

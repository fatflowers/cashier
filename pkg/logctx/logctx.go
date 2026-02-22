package logctx

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// FromGin returns a request-scoped logger from gin.Context if present,
// otherwise returns the provided base logger.
func FromGin(c *gin.Context, base *zap.SugaredLogger) *zap.SugaredLogger {
	if c == nil {
		return base
	}
	if l, ok := c.Get("logger"); ok {
		if lg, ok := l.(*zap.SugaredLogger); ok && lg != nil {
			return lg
		}
	}
	// fall back to ctx-based enrichment
	return FromCtx(c.Request.Context(), base)
}

// FromCtx returns a logger from context if set, otherwise attempts to enrich
// base with trace_id/user_id from context values.
func FromCtx(ctx context.Context, base *zap.SugaredLogger) *zap.SugaredLogger {
	if ctx == nil {
		return base
	}
	if lg, ok := ctx.Value("logger").(*zap.SugaredLogger); ok && lg != nil {
		return lg
	}
	// enrich from primitives if available
	var fields []interface{}
	if tid, ok := ctx.Value("traceID").(string); ok && tid != "" {
		fields = append(fields, "trace_id", tid)
	}
	if uid, ok := ctx.Value("user_id").(string); ok && uid != "" {
		fields = append(fields, "user_id", uid)
	}
	if len(fields) > 0 {
		return base.With(fields...)
	}
	return base
}

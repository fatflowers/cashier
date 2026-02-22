package gormlog

import (
    "context"
    "path/filepath"
    "strings"
    "time"

    "go.uber.org/zap"
    gormlogger "gorm.io/gorm/logger"
    "gorm.io/gorm/utils"

    "github.com/fatflowers/cashier/pkg/logctx"
)

// ZapLogger implements gorm.io/gorm/logger.Interface and enriches logs with
// trace_id and user_id from context via logctx.FromCtx.
type ZapLogger struct {
	base   *zap.SugaredLogger
	config gormlogger.Config
}

func New(base *zap.SugaredLogger) *ZapLogger {
	cfg := gormlogger.Config{
		SlowThreshold:             500 * time.Millisecond,
		LogLevel:                  gormlogger.Info,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	}
	return &ZapLogger{base: base, config: cfg}
}

func (z *ZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	cfg := z.config
	cfg.LogLevel = level
	return &ZapLogger{base: z.base, config: cfg}
}

func (z *ZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if z.config.LogLevel >= gormlogger.Info {
		logctx.FromCtx(ctx, z.base).Infow(msg, "args", data)
	}
}

func (z *ZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if z.config.LogLevel >= gormlogger.Warn {
		logctx.FromCtx(ctx, z.base).Warnw(msg, "args", data)
	}
}

func (z *ZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if z.config.LogLevel >= gormlogger.Error {
		logctx.FromCtx(ctx, z.base).Errorw(msg, "args", data)
	}
}

func (z *ZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if z.config.LogLevel == gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	lg := logctx.FromCtx(ctx, z.base)
    fields := []interface{}{
        "rows", rows,
        "elapsed_ms", elapsed.Milliseconds(),
        "caller", shortCaller(utils.FileWithLineNum()),
    }
	if err != nil {
		lg.Errorw("gorm_trace", append(fields, "err", err, "sql", sql)...)
		return
	}
	if z.config.SlowThreshold > 0 && elapsed > z.config.SlowThreshold {
		lg.Warnw("gorm_slow", append(fields, "sql", sql)...)
		return
	}
	if z.config.LogLevel >= gormlogger.Info {
		lg.Infow("gorm", append(fields, "sql", sql)...)
	}
}

// shortCaller trims absolute build paths to repo-relative where possible.
// Examples:
//   /Users/alex/repo/internal/platform/db/postgres.go:38 -> internal/platform/db/postgres.go:38
//   C:\repo\project\pkg\x\y.go:12 -> pkg/x/y.go:12
func shortCaller(s string) string {
    if s == "" {
        return s
    }
    // split into path and :line suffix
    pathPart := s
    linePart := ""
    if idx := strings.LastIndex(s, ":"); idx >= 0 {
        pathPart = s[:idx]
        linePart = s[idx:]
    }
    p := filepath.ToSlash(pathPart)
    // prefer well-known repo roots
    for _, marker := range []string{"/internal/", "/pkg/", "/cmd/"} {
        if i := strings.Index(p, marker); i >= 0 {
            // remove leading slash
            rel := p[i+1:]
            return rel + linePart
        }
    }
    // fallback: last 3 segments
    parts := strings.Split(p, "/")
    n := len(parts)
    if n >= 3 {
        return strings.Join(parts[n-3:], "/") + linePart
    }
    if strings.HasPrefix(p, "/") {
        p = p[1:]
    }
    return p + linePart
}

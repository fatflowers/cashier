package logger

import (
    "go.uber.org/fx"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func New() (*zap.SugaredLogger, error) {
    cfg := zap.NewProductionConfig()
    cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    cfg.EncoderConfig.TimeKey = "time"
    l, err := cfg.Build()
    if err != nil {
        return nil, err
    }
    return l.Sugar(), nil
}

var Module = fx.Options(
    fx.Provide(New),
)

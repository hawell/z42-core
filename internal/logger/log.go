package logger

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(config *Config) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	_ = level.UnmarshalText([]byte(config.Level))

	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{config.Destination},
		ErrorOutputPaths: []string{config.Destination},
	}
	return zapConfig.Build()
}

func MiddlewareFunc(logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger.Info("",
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.Request.URL.EscapedPath()),
		)
		ctx.Next()
	}
}

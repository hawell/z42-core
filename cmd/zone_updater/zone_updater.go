package main

import (
	"flag"
	"fmt"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	configPtr := flag.String("c", "config.json", "path to config file")
	flag.Parse()
	configFile := *configPtr
	config, err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	eventLoggerConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
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
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	eventLogger, err := eventLoggerConfig.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(eventLogger)

	db, err := database.Connect(config.DBConnectionString)
	if err != nil {
		zap.L().Fatal("database connection failed", zap.Error(err))
	}

	dh := storage.NewDataHandler(config.RedisData)

	for {
		revision, err := dh.GetRevision()
		if err != nil {
			zap.L().Fatal("get revision failed", zap.Error(err))
		}

		events, err := db.GetEvents(revision, 0, 100)
		if err != nil {
			zap.L().Fatal("get events failed", zap.Error(err))
		}
		if len(events) > 0 {
			zap.L().Info(fmt.Sprintf("%d new events", len(events)))
		}

		for _, event := range events {
			if err := dh.ApplyEvent(event); err != nil {
				zap.L().Fatal("apply event failed", zap.Error(err))
			}
		}

		time.Sleep(time.Second)
	}
}

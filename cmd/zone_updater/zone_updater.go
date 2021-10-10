package main

import (
	"flag"
	"fmt"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/logger"
	"github.com/hawell/z42/internal/storage"
	"go.uber.org/zap"
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

	eventLogger, err := logger.NewLogger(config.EventLog)
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

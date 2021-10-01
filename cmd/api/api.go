package main

import (
	"flag"
	"fmt"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/server"
	"github.com/hawell/z42/internal/mailer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
		panic(err)
	}

	m := mailer.NewSMTP(config.MailerConfig)

	s := server.NewServer(config.ServerConfig, db, m)
	err = s.ListenAndServer()
	fmt.Println(err)
}

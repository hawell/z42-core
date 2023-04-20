package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"z42-core/internal/api/database"
	"z42-core/internal/api/server"
	"z42-core/internal/logger"
	"z42-core/internal/mailer"
	"z42-core/internal/upstream"
)

func main() {
	configPtr := flag.String("c", "config.json", "path to config file")
	flag.Parse()
	configFile := *configPtr
	config, err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	eventLogger, err := logger.NewLogger(&config.EventLog)
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(eventLogger)

	db, err := database.Connect(config.DBConnectionString)
	if err != nil {
		panic(err)
	}

	m, err := mailer.NewSMTP(&config.MailerConfig)
	if err != nil {
		panic(err)
	}

	accessLogger, err := logger.NewLogger(&config.AccessLog)
	if err != nil {
		panic(err)
	}

	u := upstream.NewUpstream(config.UpstreamConfig)

	gin.SetMode(gin.ReleaseMode)
	s := server.NewServer(&config.ServerConfig, db, m, u, accessLogger)
	err = s.ListenAndServer()
	fmt.Println(err)
}

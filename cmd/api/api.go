package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/server"
	"github.com/hawell/z42/internal/logger"
	"github.com/hawell/z42/internal/mailer"
	"go.uber.org/zap"
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
		panic(err)
	}

	m, err := mailer.NewSMTP(config.MailerConfig)
	if err != nil {
		panic(err)
	}

	accessLogger, err := logger.NewLogger(config.AccessLog)
	if err != nil {
		panic(err)
	}

	gin.SetMode(gin.ReleaseMode)
	s := server.NewServer(config.ServerConfig, db, m, accessLogger)
	err = s.ListenAndServer()
	fmt.Println(err)
}

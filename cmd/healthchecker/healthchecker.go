package main

import (
	"flag"
	"fmt"
	"github.com/hawell/z42/internal/healthcheck"
	"github.com/hawell/z42/internal/logger"
	"github.com/hawell/z42/internal/storage"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	redisDataHandler *storage.DataHandler
	redisStatHandler *storage.StatHandler
	healthChecker    *healthcheck.Healthcheck
	configFile       string
)

func main() {
	configPtr := flag.String("c", "config.json", "path to config file")
	verifyPtr := flag.Bool("t", false, "verify configuration")
	generateConfigPtr := flag.String("g", "template-config.json", "generate template config file")

	flag.Parse()
	flagset := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagset[f.Name] = true })

	configFile = *configPtr
	if *verifyPtr {
		Verify(configFile)
		return
	}

	if flagset["g"] {
		data, err := jsoniter.MarshalIndent(DefaultConfig(), "", "  ")
		if err != nil {
			fmt.Println("cannot unmarshal template config : ", err)
			return
		}
		if err = ioutil.WriteFile(*generateConfigPtr, data, 0644); err != nil {
			fmt.Printf("cannot save template config to file %s : %s\n", *generateConfigPtr, err)
		}
		return
	}

	Start()

	// TODO: this should be part of a general api
	go func() {
		log.Println(http.ListenAndServe("localhost:6061", nil))
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP)

	for sig := range c {
		switch sig {
		case syscall.SIGINT:
			Stop()
			return
		case syscall.SIGHUP:
			Stop()
			Start()
		}
	}
}

func Start() {
	log.Printf("[INFO] loading config : %s", configFile)
	cfg, _ := LoadConfig(configFile)

	log.Printf("[INFO] loading logger...")
	accessLogger, err := logger.NewLogger(&cfg.AccessLog)
	if err != nil {
		panic(err)
	}
	eventLogger, err := logger.NewLogger(&cfg.EventLog)
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(eventLogger)
	log.Printf("[INFO] logger loaded")

	redisDataHandler = storage.NewDataHandler(&cfg.RedisData)
	redisDataHandler.Start()
	redisStatHandler = storage.NewStatHandler(&cfg.RedisStat)

	eventLogger.Info("starting health checker...")
	healthChecker = healthcheck.NewHealthcheck(&cfg.Healthcheck, redisDataHandler, redisStatHandler, accessLogger)
	eventLogger.Info("health checker started")
}

func Stop() {
	healthChecker.ShutDown()
	redisDataHandler.ShutDown()
	redisStatHandler.ShutDown()
}

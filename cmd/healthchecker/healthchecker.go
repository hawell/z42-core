package main

import (
	"flag"
	"fmt"
	"github.com/hawell/z42/internal/healthcheck"
	"github.com/hawell/z42/internal/storage"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

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
		data, err := jsoniter.MarshalIndent(healthcheckerDefaultConfig, "", "  ")
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
	requestLoggerConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			NameKey:        "logger",
			MessageKey:     "dns_healthcheck",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	requestLogger, err := requestLoggerConfig.Build()
	if err != nil {
		panic(err)
	}
	eventLoggerConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.ErrorLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
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
	log.Printf("[INFO] logger loaded")

	redisDataHandler = storage.NewDataHandler(&cfg.RedisData)
	redisStatHandler = storage.NewStatHandler(&cfg.RedisStat)

	eventLogger.Info("starting health checker...")
	healthChecker = healthcheck.NewHealthcheck(&cfg.Healthcheck, redisDataHandler, redisStatHandler, requestLogger)
	eventLogger.Info("health checker started")
}

func Stop() {
	healthChecker.ShutDown()
	redisDataHandler.ShutDown()
	redisStatHandler.ShutDown()
}

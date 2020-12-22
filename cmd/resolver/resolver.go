package main

import (
	"flag"
	"github.com/hawell/z42/internal/handler"
	"github.com/hawell/z42/internal/server"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/ratelimit"
	"github.com/json-iterator/go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/miekg/dns"
	_ "net/http/pprof"
)

var (
	servers           []*dns.Server
	redisDataHandler  *storage.DataHandler
	redisStatHandler  *storage.StatHandler
	dnsRequestHandler *handler.DnsRequestHandler
	rateLimiter       *ratelimit.RateLimiter
	configFile        string
	requestLogger     *zap.Logger
	eventLogger       *zap.Logger
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
		data, err := jsoniter.MarshalIndent(resolverDefaultConfig, "", "  ")
		if err != nil {
			log.Println("cannot unmarshal template config : ", err)
			return
		}
		if err = ioutil.WriteFile(*generateConfigPtr, data, 0644); err != nil {
			log.Printf("cannot save template config to file %s : %s\n", *generateConfigPtr, err)
		}
		return
	}

	Start()

	// TODO: this should be part of a general api
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
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

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	context := handler.NewRequestContext(w, r)
	zap.L().Debug(
		"handle request",
		zap.Uint16("id", r.Id),
		zap.String("query", context.RawName()),
		zap.String("type", context.Type()),
	)

	if rateLimiter.CanHandle(context.IP()) {
		dnsRequestHandler.HandleRequest(context)
	} else {
		context.Res = dns.RcodeRefused
		context.Response()
	}
}

func Start() {
	log.Printf("[INFO] loading config : %s", configFile)
	cfg, err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	log.Printf("[INFO] loading logger...")
	requestLoggerConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			NameKey:        "logger",
			MessageKey:     "dns_query",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	requestLogger, err = requestLoggerConfig.Build()
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
	eventLogger, err = eventLoggerConfig.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(eventLogger)
	log.Printf("[INFO] logger loaded")

	servers = server.NewServer(cfg.Server)

	redisDataHandler = storage.NewDataHandler(&cfg.RedisData)
	redisStatHandler = storage.NewStatHandler(&cfg.RedisStat)

	eventLogger.Info("starting handler...")
	dnsRequestHandler = handler.NewHandler(&cfg.Handler, redisDataHandler, requestLogger)
	eventLogger.Info("handler started")

	rateLimiter = ratelimit.NewRateLimiter(&cfg.RateLimit)

	dns.HandleFunc(".", handleRequest)

	eventLogger.Info("binding listeners...")
	for i := range servers {
		go func(i int) {
			err := servers[i].ListenAndServe()
			if err != nil {
				eventLogger.Error("listener error : %s", zap.Error(err))
			}
		}(i)
	}
	eventLogger.Info("binding completed")
}

func Stop() {
	for i := range servers {
		_ = servers[i].Shutdown()
	}
	dnsRequestHandler.ShutDown()
	redisDataHandler.ShutDown()
	redisStatHandler.ShutDown()
	requestLogger.Sync()
	eventLogger.Sync()
}

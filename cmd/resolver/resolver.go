package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"z42-core/internal/logger"
	"z42-core/internal/resolver"
	"z42-core/internal/server"
	"z42-core/internal/storage"
	"z42-core/pkg/ratelimit"
	"github.com/json-iterator/go"
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

var (
	servers           []*dns.Server
	redisDataHandler  *storage.DataHandler
	redisStatHandler  *storage.StatHandler
	dnsRequestHandler *resolver.DnsRequestHandler
	rateLimiter       *ratelimit.RateLimiter
	configFile        string
	accessLogger      *zap.Logger
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
		data, err := jsoniter.MarshalIndent(DefaultConfig(), "", "  ")
		if err != nil {
			log.Println("cannot unmarshal template config : ", err)
			return
		}
		if err = os.WriteFile(*generateConfigPtr, data, 0644); err != nil {
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
	context := resolver.NewRequestContext(w, r)
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
	accessLogger, err = logger.NewLogger(&cfg.AccessLog)
	if err != nil {
		panic(err)
	}
	eventLogger, err = logger.NewLogger(&cfg.EventLog)
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(eventLogger)
	log.Printf("[INFO] logger loaded")

	servers = server.NewServer(cfg.Server)

	redisDataHandler = storage.NewDataHandler(&cfg.RedisData)
	redisDataHandler.Start()
	redisStatHandler = storage.NewStatHandler(&cfg.RedisStat)

	eventLogger.Info("starting handler...")
	dnsRequestHandler = resolver.NewHandler(&cfg.Handler, redisDataHandler, accessLogger)
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
	_ = accessLogger.Sync()
	_ = eventLogger.Sync()
}

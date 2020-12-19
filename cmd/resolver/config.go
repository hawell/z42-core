package main

import (
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/getsentry/raven-go"
	"github.com/hawell/logger"
	"github.com/hawell/z42/internal/handler"
	"github.com/hawell/z42/internal/server"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/internal/upstream"
	"github.com/hawell/z42/pkg/geoip"
	"github.com/hawell/z42/pkg/hiredis"
	"github.com/hawell/z42/pkg/ratelimit"
	jsoniter "github.com/json-iterator/go"
	"github.com/logrusorgru/aurora"
	"github.com/miekg/dns"
	"github.com/oschwald/maxminddb-golang"
	"log"
	"log/syslog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server    []server.ServerConfig           `json:"server"`
	ErrorLog  logger.LogConfig                `json:"error_log"`
	RedisData storage.DataHandlerConfig       `json:"redis_data"`
	RedisStat storage.StatHandlerConfig       `json:"redis_stat"`
	Handler   handler.DnsRequestHandlerConfig `json:"handler"`
	RateLimit ratelimit.Config                `json:"ratelimit"`
}

var resolverDefaultConfig = &Config{
	Server: []server.ServerConfig{
		{
			Ip:       "127.0.0.1",
			Port:     1053,
			Protocol: "udp",
			Count:    1,
			Tls: server.TlsConfig{
				Enable:   false,
				CertPath: "",
				KeyPath:  "",
				CaPath:   "",
			},
		},
	},
	RedisData: storage.DataHandlerConfig{
		ZoneCacheSize:      10000,
		ZoneCacheTimeout:   60,
		ZoneReload:         60,
		RecordCacheSize:    1000000,
		RecordCacheTimeout: 60,
		Redis: hiredis.Config{
			Address:  "127.0.0.1:6379",
			Net:      "tcp",
			DB:       0,
			Password: "",
			Prefix:   "z42_",
			Suffix:   "_z42",
			Connection: hiredis.ConnectionConfig{
				MaxIdleConnections:   10,
				MaxActiveConnections: 10,
				ConnectTimeout:       500,
				ReadTimeout:          500,
				IdleKeepAlive:        30,
				MaxKeepAlive:         0,
				WaitForConnection:    false,
			},
		},
	},
	RedisStat: storage.StatHandlerConfig{
		Redis: hiredis.Config{
			Address:  "127.0.0.1:6379",
			Net:      "tcp",
			DB:       0,
			Password: "",
			Prefix:   "z42_",
			Suffix:   "_z42",
			Connection: hiredis.ConnectionConfig{
				MaxIdleConnections:   10,
				MaxActiveConnections: 10,
				ConnectTimeout:       500,
				ReadTimeout:          500,
				IdleKeepAlive:        30,
				MaxKeepAlive:         0,
				WaitForConnection:    false,
			},
		},
	},
	Handler: handler.DnsRequestHandlerConfig{
		Upstream: []upstream.UpstreamConfig{
			{
				Ip:       "1.1.1.1",
				Port:     53,
				Protocol: "udp",
				Timeout:  400,
			},
		},
		GeoIp: geoip.Config{
			Enable:    false,
			CountryDB: "geoCity.mmdb",
			ASNDB:     "geoIsp.mmdb",
		},
		MaxTtl:            3600,
		LogSourceLocation: false,
		Log: logger.LogConfig{
			Enable:     true,
			Target:     "file",
			Level:      "info",
			Path:       "/tmp/z42.log",
			Format:     "json",
			TimeFormat: time.RFC3339,
			Sentry: logger.SentryConfig{
				Enable: false,
				DSN:    "",
			},
			Syslog: logger.SyslogConfig{
				Enable:   false,
				Protocol: "tcp",
				Address:  "localhost:514",
			},
			Kafka: logger.KafkaConfig{
				Enable:      false,
				Topic:       "z42",
				Brokers:     []string{"127.0.0.1:9092"},
				Format:      "json",
				Compression: "none",
				Timeout:     3000,
				BufferSize:  1000,
			},
		},
	},
	ErrorLog: logger.LogConfig{
		Enable:     true,
		Target:     "stdout",
		Level:      "info",
		Path:       "/tmp/error.log",
		Format:     "text",
		TimeFormat: time.RFC3339,
		Sentry: logger.SentryConfig{
			Enable: false,
			DSN:    "",
		},
		Syslog: logger.SyslogConfig{
			Enable:   false,
			Protocol: "tcp",
			Address:  "locahost:514",
		},
		Kafka: logger.KafkaConfig{
			Enable:      false,
			Topic:       "z42",
			Brokers:     []string{"127.0.0.1:9092"},
			Format:      "json",
			Compression: "none",
			Timeout:     3000,
			BufferSize:  1000,
		},
	},
	RateLimit: ratelimit.Config{
		Enable:    false,
		Rate:      60,
		Burst:     10,
		BlackList: []string{},
		WhiteList: []string{},
	},
}

func LoadConfig(path string) (*Config, error) {
	config := resolverDefaultConfig
	configFile, err := os.Open(path)
	if err != nil {
		log.Printf("[ERROR] cannot load file %s : %s", path, err)
		log.Printf("[INFO] loading default config")
		return config, err
	}
	decoder := jsoniter.NewDecoder(configFile)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(config)
	if err != nil {
		log.Printf("[ERROR] cannot load json file")
		log.Printf("[INFO] loading default config")
		return config, err
	}
	return config, nil
}

func Verify(configFile string) {
	ok := aurora.Bold(aurora.Green("[ OK ]"))
	fail := aurora.Bold(aurora.Red("[FAIL]"))
	warn := aurora.Bold(aurora.Yellow("[WARN]"))
	printResult := func(msg string, err error) {
		if err == nil {
			fmt.Printf("%-60s%s\n", msg, ok)
			return
		} else {
			fmt.Printf("%-60s%s : %s\n", msg, fail, err)
		}
	}
	printWarning := func(msg string, warning string) {
		fmt.Printf("%-60s%s : %s\n", msg, warn, warning)
	}

	checkAddress := func(protocol string, ip string, port int) {
		msg := fmt.Sprintf("checking protocol : %s", protocol)
		var err error = nil
		if protocol != "tcp" && protocol != "udp" {
			err = errors.New("invalid protocol")
		}
		printResult(msg, err)

		msg = fmt.Sprintf("checking ip address : %s", ip)
		err = nil
		if ip := net.ParseIP(ip); ip == nil {
			err = errors.New("invalid ip address")
		}
		printResult(msg, err)

		msg = fmt.Sprintf("checking port number : %d", port)
		err = nil
		if port > 65535 || port < 1 {
			err = errors.New("invalid port number")
		}
		printResult(msg, err)
	}

	checkRedis := func(config *hiredis.Config) {
		fmt.Println("checking redis...")
		rd := hiredis.NewRedis(config)
		msg := fmt.Sprintf("checking whether %s://%s is available", config.Net, config.Address)
		err := rd.Ping()
		printResult(msg, err)
		msg = fmt.Sprintf("checking notify-keyspace-events")
		var nkse string
		nkse, err = rd.GetConfig("notify-keyspace-events")
		if err == nil {
			if !strings.Contains(nkse, "K") {
				err = errors.New("keyspace in not active")
			} else if !strings.Contains(nkse, "A") && !strings.Contains(nkse, "s") {
				err = errors.New("A or s should be active")
			}
		}
		printResult(msg, err)
	}

	checkLog := func(config *logger.LogConfig) {
		fmt.Println("checking log...")
		msg := fmt.Sprintf("checking target : %s", config.Path)
		var err error = nil
		if config.Target != "stdout" && config.Target != "stderr" && config.Target != "file" && config.Target != "udp" {
			err = errors.New("invalid target : " + config.Target)
		}
		printResult(msg, err)

		if config.Target == "file" {
			msg = fmt.Sprintf("checking file target : %s", config.Path)
			var file *os.File
			file, err = os.OpenFile(config.Target, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			if err == nil {
				_ = file.Close()
			}
			printResult(msg, err)
		}
		if config.Target == "udp" {
			msg = fmt.Sprintf("checking udp target : %s", config.Target)
			var raddr *net.UDPAddr
			raddr, err = net.ResolveUDPAddr("udp", config.Path)
			if err == nil {
				var con *net.UDPConn
				con, err = net.DialUDP("udp", nil, raddr)
				if err == nil {
					_ = con.Close()
				}
			}
			printResult(msg, err)
		}

		msg = fmt.Sprintf("checking log level : %s", config.Level)
		err = nil
		if config.Level != "debug" && config.Level != "info" && config.Level != "warning" && config.Level != "error" {
			err = errors.New("invalid log level : " + config.Level)
		}
		printResult(msg, err)

		msg = fmt.Sprintf("checking format : %s", config.Format)
		err = nil
		if config.Format != "text" && config.Format != "json" && config.Format != "capnp_request" {
			err = errors.New("invalid log format : " + config.Format)
		}
		printResult(msg, err)

		msg = fmt.Sprintf("checking time format : %s", config.TimeFormat)
		t1, _ := time.Parse(time.RFC3339, time.RFC3339)
		timeStr := t1.Format(config.TimeFormat)
		var t2 time.Time
		t2, err = time.Parse(config.TimeFormat, timeStr)
		if err == nil {
			if t2 != t1 {
				err = errors.New("invalid time format")
			}
		}
		printResult(msg, err)

		if config.Kafka.Enable {
			fmt.Println("checking kafka at ", config.Kafka.Brokers)
			msg = fmt.Sprintf("checking kafka")
			cfg := sarama.NewConfig()
			cfg.Producer.RequiredAcks = sarama.WaitForAll
			cfg.Producer.Compression = sarama.CompressionNone
			cfg.Producer.Flush.Frequency = 500 * time.Millisecond
			cfg.Producer.Return.Errors = true
			cfg.Producer.Return.Successes = true

			cfg.Metadata.Timeout = time.Duration(config.Kafka.Timeout) * time.Millisecond

			var producer sarama.SyncProducer
			producerMessages := []*sarama.ProducerMessage{
				{
					Topic:    config.Kafka.Topic,
					Value:    sarama.StringEncoder("test message"),
					Metadata: "test",
				},
			}
			producer, err = sarama.NewSyncProducer(config.Kafka.Brokers, cfg)
			if err == nil {
				err = producer.SendMessages(producerMessages)
			}
			printResult(msg, err)
		}
		if config.Sentry.Enable {
			msg = fmt.Sprintf("checking sentry at %s", config.Sentry.DSN)
			var client *raven.Client
			client, err = raven.New(config.Sentry.DSN)
			if err == nil {
				packet := raven.NewPacket("test message", nil)
				eventID, ch := client.Capture(packet, nil)
				if eventID != "" {
					err = <-ch
				}
				if err == nil && eventID == "" {
					err = errors.New("sentry test failed")
				}
			}
			printResult(msg, err)
		}
		if config.Syslog.Enable {
			msg = fmt.Sprintf("checking syslog at %s", config.Syslog.Address)
			var w *syslog.Writer
			w, err = syslog.Dial(config.Syslog.Protocol, config.Syslog.Address, syslog.LOG_ERR, "syslog test")
			if err == nil {
				err = w.Err("test message")
			}
			printResult(msg, err)
		}
	}

	fmt.Println("Starting Config Verification")

	msg := fmt.Sprintf("loading config file : %s", configFile)
	config, err := LoadConfig(configFile)
	printResult(msg, err)

	fmt.Println("checking listeners...")
	for _, serverConfig := range config.Server {
		checkAddress(serverConfig.Protocol, serverConfig.Ip, serverConfig.Port)
		msg = fmt.Sprintf("checking port number : %d", serverConfig.Port)
		if serverConfig.Port != 53 {
			printWarning(msg, "using non-standard port")
		} else {
			printResult(msg, nil)
		}

		address := serverConfig.Ip + ":" + strconv.Itoa(serverConfig.Port)
		msg = fmt.Sprintf("checking whether %s://%s is available", serverConfig.Protocol, address)
		if serverConfig.Protocol == "udp" {
			var ln net.PacketConn
			ln, err = net.ListenPacket(serverConfig.Protocol, address)
			if err == nil {
				_ = ln.Close()
			}
		} else {
			var ln net.Listener
			ln, err = net.Listen(serverConfig.Protocol, address)
			if err == nil {
				_ = ln.Close()
			}
		}
		printResult(msg, err)
	}
	fmt.Println("checking upstreams...")
	for _, upstreamConfig := range config.Handler.Upstream {
		checkAddress(upstreamConfig.Protocol, upstreamConfig.Ip, upstreamConfig.Port)
		address := upstreamConfig.Ip + ":" + strconv.Itoa(upstreamConfig.Port)
		msg = fmt.Sprintf("checking whether %s://%s is available", upstreamConfig.Protocol, address)
		client := &dns.Client{
			Net:     upstreamConfig.Protocol,
			Timeout: time.Duration(upstreamConfig.Timeout) * time.Millisecond,
		}
		m := new(dns.Msg)
		m.SetQuestion("dns.msftncsi.com.", dns.TypeA)
		var resp *dns.Msg
		resp, _, err = client.Exchange(m, address)
		if err == nil {
			if len(resp.Answer) == 0 {
				err = errors.New("empty response")
			} else {
				a, ok := resp.Answer[0].(*dns.A)
				if !ok {
					err = errors.New("bad response")
				} else if a.A.String() != "131.107.255.255" {
					err = errors.New("incorrect response")
				}
			}
		}
		printResult(msg, err)
	}
	checkRedis(&config.RedisData.Redis)
	checkRedis(&config.RedisStat.Redis)
	if config.Handler.GeoIp.Enable {
		fmt.Println("checking geoip...")
		var countryRecord struct {
			Location struct {
				Latitude        float64 `maxminddb:"latitude"`
				LongitudeOffset uintptr `maxminddb:"longitude"`
			} `maxminddb:"location"`
			Country struct {
				ISOCode string `maxminddb:"iso_code"`
			} `maxminddb:"country"`
		}
		var asnRecord struct {
			AutonomousSystemNumber uint `maxminddb:"autonomous_system_number"`
		}
		records := []interface{}{countryRecord, asnRecord}
		for i, dbFile := range []string{config.Handler.GeoIp.CountryDB, config.Handler.GeoIp.ASNDB} {
			msg = fmt.Sprintf("checking file stat : %s", dbFile)
			_, err = os.Stat(dbFile)
			printResult(msg, err)
			if err == nil {
				msg = fmt.Sprintf("checking db : %s", dbFile)
				var db *maxminddb.Reader
				db, err = maxminddb.Open(dbFile)
				printResult(msg, err)
				if err == nil {
					msg = fmt.Sprintf("checking db query results")
					err = db.Lookup(net.ParseIP("46.19.36.12"), &records[i])
					printResult(msg, err)
				}
			}
		}
	}
	if config.ErrorLog.Enable {
		checkLog(&config.ErrorLog)
	}
	if config.Handler.Log.Enable {
		checkLog(&config.Handler.Log)
	}
}

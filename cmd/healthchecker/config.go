package main

import (
	"github.com/hawell/logger"
	"github.com/hawell/z42/internal/healthcheck"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/hiredis"
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
	"time"
)

type Config struct {
	ErrorLog    logger.LogConfig          `json:"error_log"`
	RedisData   storage.DataHandlerConfig `json:"redis_data"`
	RedisStat   storage.StatHandlerConfig `json:"redis_stat"`
	Healthcheck healthcheck.Config        `json:"healthcheck"`
}

var healthcheckerDefaultConfig = &Config{
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
	Healthcheck: healthcheck.Config{
		Enable:             false,
		MaxRequests:        10,
		MaxPendingRequests: 100,
		UpdateInterval:     600,
		CheckInterval:      600,
		Log: logger.LogConfig{
			Enable:     true,
			Target:     "file",
			Level:      "info",
			Path:       "/tmp/healthcheck.log",
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
}

func LoadConfig(path string) (*Config, error) {
	config := healthcheckerDefaultConfig
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

func Verify(_ string) {

}

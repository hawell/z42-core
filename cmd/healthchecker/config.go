package main

import (
	"github.com/hawell/z42/internal/healthcheck"
	"github.com/hawell/z42/internal/logger"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/hiredis"
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
)

type Config struct {
	AccessLog   *logger.Config            `json:"access_log"`
	EventLog    *logger.Config            `json:"event_log"`
	RedisData   storage.DataHandlerConfig `json:"redis_data"`
	RedisStat   storage.StatHandlerConfig `json:"redis_stat"`
	Healthcheck healthcheck.Config        `json:"healthcheck"`
}

var healthcheckerDefaultConfig = &Config{
	AccessLog: &logger.Config{
		Level:       "INFO",
		Destination: "stdout",
	},
	EventLog: &logger.Config{
		Level:       "ERROR",
		Destination: "stderr",
	},
	RedisData: storage.DataHandlerConfig{
		ZoneCacheSize:      10000,
		ZoneCacheTimeout:   60,
		ZoneReload:         60,
		RecordCacheSize:    1000000,
		RecordCacheTimeout: 60,
		MinTTL:             5,
		MaxTTL:             300,
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

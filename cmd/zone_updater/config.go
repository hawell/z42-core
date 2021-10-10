package main

import (
	"github.com/hawell/z42/internal/logger"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/hiredis"
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
)

type Config struct {
	EventLog           *logger.Config             `json:"event_log"`
	DBConnectionString string                     `json:"db_connection_string"`
	RedisData          *storage.DataHandlerConfig `json:"redis_data"`
}

var apiDefaultConfig = &Config{
	EventLog: &logger.Config{
		Level:       "error",
		Destination: "stderr",
	},
	DBConnectionString: "root:root@tcp(127.0.0.1:3306)/z42",
	RedisData: &storage.DataHandlerConfig{
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
}

func LoadConfig(path string) (*Config, error) {
	config := apiDefaultConfig
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

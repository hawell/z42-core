package main

import (
	"log"
	"os"

	jsoniter "github.com/json-iterator/go"
	"z42-core/internal/logger"
	"z42-core/internal/storage"
)

type Config struct {
	EventLog           logger.Config             `json:"event_log"`
	DBConnectionString string                    `json:"db_connection_string"`
	RedisData          storage.DataHandlerConfig `json:"redis_data"`
}

func DefaultConfig() Config {
	return Config{
		EventLog:           logger.DefaultConfig(),
		DBConnectionString: "root:root@tcp(127.0.0.1:3306)/z42",
		RedisData:          storage.DefaultDataHandlerConfig(),
	}
}

func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()
	configFile, err := os.Open(path)
	if err != nil {
		log.Printf("[ERROR] cannot load file %s : %s", path, err)
		log.Printf("[INFO] loading default config")
		return &config, err
	}
	decoder := jsoniter.NewDecoder(configFile)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&config)
	if err != nil {
		log.Printf("[ERROR] cannot load json file")
		log.Printf("[INFO] loading default config")
		return &config, err
	}
	return &config, nil
}

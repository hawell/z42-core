package main

import (
	"fmt"
	"github.com/hawell/z42/configs"
	"github.com/hawell/z42/internal/healthcheck"
	"github.com/hawell/z42/internal/logger"
	"github.com/hawell/z42/internal/storage"
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
)

type Config struct {
	AccessLog   logger.Config             `json:"access_log"`
	EventLog    logger.Config             `json:"event_log"`
	RedisData   storage.DataHandlerConfig `json:"redis_data"`
	RedisStat   storage.StatHandlerConfig `json:"redis_stat"`
	Healthcheck healthcheck.Config        `json:"healthcheck"`
}

func DefaultConfig() Config {
	return Config{
		AccessLog:   logger.DefaultConfig(),
		EventLog:    logger.DefaultConfig(),
		RedisData:   storage.DefaultDataHandlerConfig(),
		RedisStat:   storage.DefaultStatHandlerConfig(),
		Healthcheck: healthcheck.DefaultConfig(),
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
	err = decoder.Decode(config)
	if err != nil {
		log.Printf("[ERROR] cannot load json file")
		log.Printf("[INFO] loading default config")
		return &config, err
	}
	return &config, nil
}

func Verify(configFile string) {
	fmt.Println("Starting Config Verification")

	msg := fmt.Sprintf("loading config file : %s", configFile)
	config, err := LoadConfig(configFile)
	configs.PrintResult(msg, err)

	fmt.Println("checking healthcheck...")
	config.Healthcheck.Verify()

	config.RedisData.Verify()
	config.RedisStat.Verify()

	config.AccessLog.Verify()
	config.EventLog.Verify()
}

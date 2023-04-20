package main

import (
	"fmt"
	"log"
	"os"

	jsoniter "github.com/json-iterator/go"
	"z42-core/configs"
	"z42-core/internal/logger"
	"z42-core/internal/resolver"
	"z42-core/internal/server"
	"z42-core/internal/storage"
	"z42-core/pkg/ratelimit"
)

type Config struct {
	Server    []server.Config           `json:"server"`
	RedisData storage.DataHandlerConfig `json:"redis_data"`
	RedisStat storage.StatHandlerConfig `json:"redis_stat"`
	Handler   resolver.Config           `json:"handler"`
	RateLimit ratelimit.Config          `json:"ratelimit"`
	EventLog  logger.Config             `json:"event_log"`
	AccessLog logger.Config             `json:"access_log"`
}

func DefaultConfig() *Config {
	return &Config{
		Server:    []server.Config{server.DefaultConfig()},
		RedisData: storage.DefaultDataHandlerConfig(),
		RedisStat: storage.DefaultStatHandlerConfig(),
		Handler:   resolver.DefaultDnsRequestHandlerConfig(),
		RateLimit: ratelimit.DefaultConfig(),
		EventLog:  logger.DefaultConfig(),
		AccessLog: logger.DefaultConfig(),
	}
}

func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()
	configFile, err := os.Open(path)
	if err != nil {
		log.Printf("[ERROR] cannot load file %s : %s", path, err)
		log.Printf("[INFO] loading default config")
		return config, err
	}
	decoder := jsoniter.NewDecoder(configFile)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&config)
	if err != nil {
		log.Printf("[ERROR] cannot load json file")
		log.Printf("[INFO] loading default config")
		return config, err
	}
	return config, nil
}

func Verify(configFile string) {
	fmt.Println("Starting Config Verification")

	msg := fmt.Sprintf("loading config file : %s", configFile)
	config, err := LoadConfig(configFile)
	configs.PrintResult(msg, err)

	fmt.Println("checking listeners...")
	for _, serverConfig := range config.Server {
		serverConfig.Verify()
	}

	fmt.Println("checking handler...")
	config.Handler.Verify()

	config.RedisData.Verify()
	config.RedisStat.Verify()

	config.AccessLog.Verify()
	config.EventLog.Verify()
}

package main

import (
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
	"z42-core/internal/api/server"
	"z42-core/internal/logger"
	"z42-core/internal/mailer"
	"z42-core/internal/upstream"
)

type Config struct {
	DBConnectionString string            `json:"db_connection_string"`
	EventLog           logger.Config     `json:"event_log"`
	AccessLog          logger.Config     `json:"access_log"`
	ServerConfig       server.Config     `json:"server"`
	MailerConfig       mailer.Config     `json:"mailer"`
	UpstreamConfig     []upstream.Config `json:"upstream"`
}

func DefaultConfig() Config {
	return Config{
		DBConnectionString: "root:root@tcp(127.0.0.1:3306)/z42",
		EventLog:           logger.DefaultConfig(),
		AccessLog:          logger.DefaultConfig(),
		ServerConfig:       server.DefaultConfig(),
		MailerConfig:       mailer.DefaultConfig(),
		UpstreamConfig:     []upstream.Config{upstream.DefaultConfig()},
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

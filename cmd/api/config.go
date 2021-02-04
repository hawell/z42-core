package main

import (
	"github.com/hawell/z42/internal/api/server"
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
)

type Config struct {
	DBConnectionString string   `json:"db_connection_string"`
	ServerConfig *server.Config `json:"server"`
}

var apiDefaultConfig = &Config{
	DBConnectionString: "root:root@tcp(127.0.0.1:3306)/z42",
	ServerConfig:       &server.Config{
		BindAddress:  "localhost:8080",
		ReadTimeout:  10,
		WriteTimeout: 10,
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


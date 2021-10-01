package main

import (
	"github.com/hawell/z42/internal/api/server"
	"github.com/hawell/z42/internal/mailer"
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
)

type Config struct {
	DBConnectionString string          `json:"db_connection_string"`
	ServerConfig       *server.Config  `json:"server"`
	MailerConfig       *mailer.Config `json:"mailer"`
}

var apiDefaultConfig = &Config{
	DBConnectionString: "root:root@tcp(127.0.0.1:3306)/z42",
	ServerConfig: &server.Config{
		BindAddress:   "localhost:8080",
		ReadTimeout:   10,
		WriteTimeout:  10,
		WebServer:    "www.z42.com",
		ApiServer:    "api.z42.com",
		NameServer:    "ns.z42.com.",
		HtmlTemplates: "./templates/*.tmpl",
	},
	MailerConfig: &mailer.Config{
		Address:   "127.0.0.1:25",
		FromName:  "z42",
		FromEmail: "noreply@z42.com",
		WebServer:    "www.z42.com",
		ApiServer:    "api.z42.com",
		NameServer:    "ns.z42.com.",
		HtmlTemplates: "./templates/*.tmpl",
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

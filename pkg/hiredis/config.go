package hiredis

import (
	"errors"
	"fmt"
	"github.com/hawell/z42/configs"
	"strings"
)

type ConnectionConfig struct {
	MaxIdleConnections   int  `json:"max_idle_connections"`
	MaxActiveConnections int  `json:"max_active_connections"`
	ConnectTimeout       int  `json:"connect_timeout"`
	ReadTimeout          int  `json:"read_timeout"`
	IdleKeepAlive        int  `json:"idle_keep_alive"`
	MaxKeepAlive         int  `json:"max_keep_alive"`
	WaitForConnection    bool `json:"wait_for_connection"`
}

type Config struct {
	Address    string           `json:"address"`
	Net        string           `json:"net"`
	DB         int              `json:"db"`
	Password   string           `json:"password"`
	Prefix     string           `json:"prefix"`
	Suffix     string           `json:"suffix"`
	Connection ConnectionConfig `json:"connection"`
}

func (c Config) Verify() {
	fmt.Println("checking redis...")
	rd := NewRedis(&c)
	msg := fmt.Sprintf("checking whether %s://%s is available", c.Net, c.Address)
	err := rd.Ping()
	configs.PrintResult(msg, err)
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
	configs.PrintResult(msg, err)
}

func DefaultConfig() Config {
	return Config{
		Address:  "127.0.0.1:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "z42_",
		Suffix:   "_z42",
		Connection: ConnectionConfig{
			MaxIdleConnections:   10,
			MaxActiveConnections: 10,
			ConnectTimeout:       500,
			ReadTimeout:          500,
			IdleKeepAlive:        30,
			MaxKeepAlive:         0,
			WaitForConnection:    false,
		},
	}
}

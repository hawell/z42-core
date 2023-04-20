package server

import (
	"fmt"
	"z42-core/configs"
	"net"
	"strconv"
)

type TlsConfig struct {
	Enable   bool   `json:"enable"`
	CertPath string `json:"cert_path"`
	KeyPath  string `json:"key_path"`
	CaPath   string `json:"ca_path"`
}

type Config struct {
	Ip       string    `json:"ip"`
	Port     int       `json:"port"`
	Protocol string    `json:"protocol"`
	Count    int       `json:"count"`
	Tls      TlsConfig `json:"tls"`
}

func DefaultConfig() Config {
	return Config{
		Ip:       "127.0.0.1",
		Port:     1053,
		Protocol: "udp",
		Count:    1,
		Tls: TlsConfig{
			Enable:   false,
			CertPath: "",
			KeyPath:  "",
			CaPath:   "",
		},
	}
}

func (c Config) Verify() {
	configs.CheckAddress(c.Protocol, c.Ip, c.Port)
	msg := fmt.Sprintf("checking port number : %d", c.Port)
	if c.Port != 53 {
		configs.PrintWarning(msg, "using non-standard port")
	} else {
		configs.PrintResult(msg, nil)
	}

	address := c.Ip + ":" + strconv.Itoa(c.Port)
	msg = fmt.Sprintf("checking whether %s://%s is available", c.Protocol, address)
	var err error
	if c.Protocol == "udp" {
		var con net.PacketConn
		con, err = net.ListenPacket(c.Protocol, address)
		if err == nil {
			_ = con.Close()
		}
	} else {
		var ln net.Listener
		ln, err = net.Listen(c.Protocol, address)
		if err == nil {
			_ = ln.Close()
		}
	}
	configs.PrintResult(msg, err)

}

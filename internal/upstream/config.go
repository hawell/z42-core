package upstream

import (
	"errors"
	"fmt"
	"z42-core/configs"
	"github.com/miekg/dns"
	"strconv"
	"time"
)

type Config struct {
	Ip       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Timeout  int    `json:"timeout"`
}

func DefaultConfig() Config {
	return Config{
		Ip:       "1.1.1.1",
		Port:     53,
		Protocol: "udp",
		Timeout:  400,
	}
}

func (c Config) Verify() {
	configs.CheckAddress(c.Protocol, c.Ip, c.Port)
	address := c.Ip + ":" + strconv.Itoa(c.Port)
	msg := fmt.Sprintf("checking whether %s://%s is available", c.Protocol, address)
	client := &dns.Client{
		Net:     c.Protocol,
		Timeout: time.Duration(c.Timeout) * time.Millisecond,
	}
	m := new(dns.Msg)
	m.SetQuestion("dns.msftncsi.com.", dns.TypeA)
	var (
		resp *dns.Msg
		err  error
	)
	resp, _, err = client.Exchange(m, address)
	if err == nil {
		if len(resp.Answer) == 0 {
			err = errors.New("empty response")
		} else {
			a, ok := resp.Answer[0].(*dns.A)
			if !ok {
				err = errors.New("bad response")
			} else if a.A.String() != "131.107.255.255" {
				err = errors.New("incorrect response")
			}
		}
	}
	configs.PrintResult(msg, err)
}

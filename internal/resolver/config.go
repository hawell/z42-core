package resolver

import (
	"encoding/hex"
	"errors"
	"fmt"
	"z42-core/configs"
	"z42-core/internal/upstream"
	"z42-core/pkg/geoip"
)

type Config struct {
	Upstream          []upstream.Config `json:"upstream"`
	GeoIp             geoip.Config      `json:"geoip"`
	CookieSecret      string            `json:"cookie_secret"`
	LogSourceLocation bool              `json:"log_source_location"`
}

func DefaultDnsRequestHandlerConfig() Config {
	return Config{
		Upstream:          []upstream.Config{upstream.DefaultConfig()},
		GeoIp:             geoip.DefaultConfig(),
		LogSourceLocation: false,
		CookieSecret:      "000102030405060708090a0b0c0d0e0f",
	}
}

func (c Config) Verify() {
	fmt.Println("checking upstreams...")
	for _, upstreamConfig := range c.Upstream {
		upstreamConfig.Verify()
	}
	c.GeoIp.Verify()
	b, err := hex.DecodeString(c.CookieSecret)
	if err == nil {
		if len(b) != 16 {
			err = errors.New("incorrect secret size: must be 16 bytes")
		}
	}
	configs.PrintResult("checking cookie secret", err)
}

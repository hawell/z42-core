package resolver

import (
	"fmt"
	"z42-core/internal/upstream"
	"z42-core/pkg/geoip"
)

type Config struct {
	Upstream          []upstream.Config `json:"upstream"`
	GeoIp             geoip.Config      `json:"geoip"`
	LogSourceLocation bool              `json:"log_source_location"`
}

func DefaultDnsRequestHandlerConfig() Config {
	return Config{
		Upstream:          []upstream.Config{upstream.DefaultConfig()},
		GeoIp:             geoip.DefaultConfig(),
		LogSourceLocation: false,
	}
}

func (c Config) Verify() {
	fmt.Println("checking upstreams...")
	for _, upstreamConfig := range c.Upstream {
		upstreamConfig.Verify()
	}
	c.GeoIp.Verify()
}

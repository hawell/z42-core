package storage

import "z42-core/pkg/hiredis"

type DataHandlerConfig struct {
	ZoneCacheSize      int            `json:"zone_cache_size"`
	ZoneCacheTimeout   int64          `json:"zone_cache_timeout"`
	ZoneReload         int            `json:"zone_reload"`
	RecordCacheSize    int            `json:"record_cache_size"`
	RecordCacheTimeout int64          `json:"record_cache_timeout"`
	Redis              hiredis.Config `json:"redis"`
	MinTTL             uint32         `json:"min_ttl"`
	MaxTTL             uint32         `json:"max_ttl"`
}

func DefaultDataHandlerConfig() DataHandlerConfig {
	return DataHandlerConfig{
		ZoneCacheSize:      10000,
		ZoneCacheTimeout:   60,
		ZoneReload:         60,
		RecordCacheSize:    1000000,
		RecordCacheTimeout: 60,
		MinTTL:             5,
		MaxTTL:             300,
		Redis:              hiredis.DefaultConfig(),
	}
}

func (c DataHandlerConfig) Verify() {
	c.Redis.Verify()
}

type StatHandlerConfig struct {
	Redis hiredis.Config `json:"redis"`
}

func DefaultStatHandlerConfig() StatHandlerConfig {
	return StatHandlerConfig{
		Redis: hiredis.DefaultConfig(),
	}
}

func (c StatHandlerConfig) Verify() {
	c.Redis.Verify()
}

package handler

import (
	"errors"
	"fmt"
	"github.com/hawell/logger"
	"github.com/hawell/redins/redis"
	"github.com/hawell/redins/test"
	"github.com/hawell/redins/upstream"
	"testing"
)

type TestCase struct {
	Name            string
	Description     string
	Enabled         bool
	RedisDataConfig redis.DataHandlerConfig
	HandlerConfig   DnsRequestHandlerConfig
	Initialize      func(testCase *TestCase) (*DnsRequestHandler, error)
	ApplyAndVerify  func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T)
	Zones           []string
	ZoneConfigs     []string
	Entries         [][][]string
	TestCases       []test.Case
}

func DefaultInitialize(testCase *TestCase) (*DnsRequestHandler, error) {
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)

	r := redis.NewDataHandler(&testCase.RedisDataConfig)
	h := NewHandler(&testCase.HandlerConfig, r)
	if err := h.RedisData.Redis.Del("*"); err != nil {
		return nil, err
	}
	for i, zone := range testCase.Zones {
		if err := h.RedisData.Redis.SAdd("redins:zones", zone); err != nil {
			return nil, err
		}
		for _, cmd := range testCase.Entries[i] {
			err := h.RedisData.Redis.HSet("redins:zones:"+zone, cmd[0], cmd[1])
			if err != nil {
				return nil, errors.New(fmt.Sprintf("[ERROR] cannot connect to redis: %s", err))
			}
		}
		if err := h.RedisData.Redis.Set("redins:zones:"+zone+":config", testCase.ZoneConfigs[i]); err != nil {
			return nil, err
		}
	}
	h.RedisData.LoadZones()
	return h, nil
}

func DefaultApplyAndVerify(testCase *TestCase, requestHandler *DnsRequestHandler, t *testing.T) {
	for i, tc := range testCase.TestCases {

		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		requestHandler.HandleRequest(state)

		resp := w.Msg

		if err := test.SortAndCheck(resp, tc); err != nil {
			fmt.Println(tc.Desc)
			fmt.Println(i, err, tc.Qname)
			fmt.Println(tc.Answer, resp.Answer)
			fmt.Println(tc.Ns, resp.Ns)
			fmt.Println(tc.Extra, resp.Extra)
			t.Fail()
		}
	}
}

var DefaultRedisDataTestConfig = redis.DataHandlerConfig{
	ZoneCacheSize:      10000,
	ZoneCacheTimeout:   60,
	ZoneReload:         60,
	RecordCacheSize:    1000000,
	RecordCacheTimeout: 60,
	Redis: redis.RedisConfig{
		Address:  "redis:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "test_",
		Suffix:   "_test",
		Connection: redis.RedisConnectionConfig{
			MaxIdleConnections:   10,
			MaxActiveConnections: 10,
			ConnectTimeout:       500,
			ReadTimeout:          500,
			IdleKeepAlive:        30,
			MaxKeepAlive:         0,
			WaitForConnection:    true,
		},
	},
}

var DefaultHandlerTestConfig = DnsRequestHandlerConfig{
	MaxTtl: 3600,
	Log: logger.LogConfig{
		Enable: false,
	},
	Upstream: []upstream.UpstreamConfig{
		{
			Ip:       "1.1.1.1",
			Port:     53,
			Protocol: "udp",
			Timeout:  1000,
		},
	},
	GeoIp: GeoIpConfig{
		Enable:    true,
		CountryDB: "../geoCity.mmdb",
		ASNDB:     "../geoIsp.mmdb",
	},
}

func CenterText(s string, w int) string {
	return fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

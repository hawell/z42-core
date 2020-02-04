package handler

import (
	"arvancloud/redins/test"
	"errors"
	"fmt"
	"github.com/hawell/logger"
	"github.com/hawell/uperdis"
	"testing"
)

type TestCase struct {
	Name           string
	Description    string
	Enabled        bool
	Config         DnsRequestHandlerConfig
	Initialize     func(testCase *TestCase) (*DnsRequestHandler, error)
	ApplyAndVerify func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T)
	Zones          []string
	ZoneConfigs    []string
	Entries        [][][]string
	TestCases      []test.Case
}

func DefaultInitialize(testCase *TestCase) (*DnsRequestHandler, error) {
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)

	h := NewHandler(&testCase.Config)
	if err := h.Redis.Del("*"); err != nil {
		return nil, err
	}
	for i, zone := range testCase.Zones {
		if err := h.Redis.SAdd("redins:zones", zone); err != nil {
			return nil, err
		}
		for _, cmd := range testCase.Entries[i] {
			err := h.Redis.HSet("redins:zones:"+zone, cmd[0], cmd[1])
			if err != nil {
				return nil, errors.New(fmt.Sprintf("[ERROR] cannot connect to redis: %s", err))
			}
		}
		if err := h.Redis.Set("redins:zones:"+zone+":config", testCase.ZoneConfigs[i]); err != nil {
			return nil, err
		}
	}
	h.LoadZones()
	return h, nil
}

func DefaultApplyAndVerify(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
	for i, tc := range testCase.TestCases {

		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		handler.HandleRequest(state)

		resp := w.Msg

		if err := test.SortAndCheck(resp, tc); err != nil {
			fmt.Println(tc.Desc)
			fmt.Println(i, err, tc.Qname, tc.Answer, resp.Answer)
			t.Fail()
		}
	}
}

var DefaultTestConfig = DnsRequestHandlerConfig{
	MaxTtl:       300,
	CacheTimeout: 60,
	ZoneReload:   600,
	Redis: uperdis.RedisConfig{
		Address:  "redis:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "test_",
		Suffix:   "_test",
		Connection: uperdis.RedisConnectionConfig{
			MaxIdleConnections:   10,
			MaxActiveConnections: 10,
			ConnectTimeout:       500,
			ReadTimeout:          500,
			IdleKeepAlive:        30,
			MaxKeepAlive:         0,
			WaitForConnection:    true,
		},
	},
	Log: logger.LogConfig{
		Enable: false,
	},
	Upstream: []UpstreamConfig{
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


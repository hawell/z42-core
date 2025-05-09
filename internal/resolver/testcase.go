package resolver

import (
	"errors"
	"fmt"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"testing"
	"z42-core/internal/storage"
	"z42-core/internal/test"
	"z42-core/internal/upstream"
	"z42-core/pkg/geoip"
	"z42-core/pkg/hiredis"
)

type TestCase struct {
	Name            string
	Description     string
	Enabled         bool
	RedisDataConfig storage.DataHandlerConfig
	HandlerConfig   Config
	Initialize      func(testCase *TestCase) (*DnsRequestHandler, error)
	ApplyAndVerify  func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T)
	Zones           []string
	ZoneConfigs     []string
	Entries         [][][]string
	TestCases       []test.Case
}

func DefaultInitialize(testCase *TestCase) (*DnsRequestHandler, error) {
	r := storage.NewDataHandler(&testCase.RedisDataConfig)
	r.Start()
	l, _ := zap.NewProduction()
	h := NewHandler(&testCase.HandlerConfig, r, l)
	if err := h.RedisData.Clear(); err != nil {
		return nil, err
	}
	for i, zone := range testCase.Zones {
		if err := r.EnableZone(zone); err != nil {
			return nil, err
		}
		for _, cmd := range testCase.Entries[i] {
			err := r.SetLocationFromJson(zone, cmd[0], cmd[1])
			if err != nil {
				return nil, errors.New(fmt.Sprintf("[ERROR] 4: %s\n%s", err, cmd[1]))
			}
		}
		if err := r.SetZoneConfigFromJson(zone, testCase.ZoneConfigs[i]); err != nil {
			return nil, err
		}
	}
	h.RedisData.LoadZones()
	return h, nil
}

func DefaultApplyAndVerify(testCase *TestCase, requestHandler *DnsRequestHandler, t *testing.T) {
	RegisterTestingT(t)
	for _, tc := range testCase.TestCases {
		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		requestHandler.HandleRequest(state)

		resp := w.Msg

		err := test.SortAndCheck(resp, tc)
		Expect(err).To(BeNil())
	}
}

var DefaultRedisDataTestConfig = storage.DataHandlerConfig{
	ZoneCacheSize:      10000,
	ZoneCacheTimeout:   60,
	ZoneReload:         60,
	RecordCacheSize:    1000000,
	RecordCacheTimeout: 60,
	MinTTL:             5,
	MaxTTL:             3600,
	Redis: hiredis.Config{
		Address:  "127.0.0.1:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "test_",
		Suffix:   "_test",
		Connection: hiredis.ConnectionConfig{
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

var DefaultHandlerTestConfig = Config{
	Upstream: []upstream.Config{
		{
			Ip:       "1.1.1.1",
			Port:     53,
			Protocol: "udp",
			Timeout:  1000,
		},
	},
	GeoIp: geoip.Config{
		Enable:    true,
		CountryDB: "../../assets/geoCity.mmdb",
		ASNDB:     "../../assets/geoIsp.mmdb",
	},
	CookieSecret: "000102030405060708090a0b0c0d0e0f",
}

func CenterText(s string, w int) string {
	return fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

package handler

import (
	"github.com/hawell/logger"
	"github.com/hawell/z42/redis"
	"github.com/hawell/z42/test"
	"github.com/miekg/dns"
	"log"
	"os"
	"testing"
)

var benchZone = "bench.zon."

var benchEntries = [][]string{
	{
		"www",
		`{
				"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]},
				"aaaa":{"ttl":300, "records":[{"ip":"::1"}]},
			}`,
	},
	{
		"www2",
		`{
				"cname":{"ttl":300, "host":"www.bench.zon."},
			}`,
	},
}

var benchTestHandler *DnsRequestHandler

func TestMain(m *testing.M) {
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)

	r := redis.NewDataHandler(&DefaultRedisDataTestConfig)
	benchTestHandler = NewHandler(&DefaultHandlerTestConfig, r)
	err := benchTestHandler.RedisData.Clear()
	log.Println(err)
	err = benchTestHandler.RedisData.EnableZone(benchZone)
	log.Println(err)
	err = benchTestHandler.RedisData.SetZoneConfigFromJson(benchZone, "{\"cname_flattening\": false}")
	log.Println(err)
	for _, cmd := range benchEntries {
		err := benchTestHandler.RedisData.SetLocationFromJson(benchZone, cmd[0], cmd[1])
		if err != nil {
			log.Printf("[ERROR] cannot connect to redis: %s", err)
			return
		}
	}

	benchTestHandler.RedisData.LoadZones()
	os.Exit(m.Run())
}

var response dns.Msg

func BenchmarkA(b *testing.B) {
	tc := test.Case{
		Qname: "www.bench.zon.", Qtype: dns.TypeA,
	}
	var resp *dns.Msg
	for n := 0; n < b.N; n++ {
		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		benchTestHandler.HandleRequest(state)

		resp = w.Msg
	}
	response = *resp
}

func BenchmarkAAAA(b *testing.B) {
	tc := test.Case{
		Qname: "www.bench.zon.", Qtype: dns.TypeAAAA,
	}
	var resp *dns.Msg
	for n := 0; n < b.N; n++ {
		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		benchTestHandler.HandleRequest(state)

		resp = w.Msg
	}
	response = *resp
}

func BenchmarkCNAME(b *testing.B) {
	tc := test.Case{
		Qname: "www2.bench.zon.", Qtype: dns.TypeA,
	}
	var resp *dns.Msg
	for n := 0; n < b.N; n++ {
		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		benchTestHandler.HandleRequest(state)

		resp = w.Msg
	}
	response = *resp
}

func BenchmarkNXDomain(b *testing.B) {
	tc := test.Case{
		Qname: "www3.bench.zon.", Qtype: dns.TypeA,
	}
	var resp *dns.Msg
	for n := 0; n < b.N; n++ {
		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		benchTestHandler.HandleRequest(state)

		resp = w.Msg
	}
	response = *resp
}

func BenchmarkNotAuth(b *testing.B) {
	tc := test.Case{
		Qname: "www.bench2.zon.", Qtype: dns.TypeA,
	}
	var resp *dns.Msg
	for n := 0; n < b.N; n++ {
		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		benchTestHandler.HandleRequest(state)

		resp = w.Msg
	}
	response = *resp
}

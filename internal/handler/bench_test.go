package handler

import (
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/internal/test"
	"github.com/miekg/dns"
	"go.uber.org/zap"
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
				"aaaa":{"ttl":300, "records":[{"ip":"::1"}]}
			}`,
	},
	{
		"www2",
		`{
				"cname":{"ttl":300, "host":"www.bench.zon."}
			}`,
	},
}

var benchTestHandler *DnsRequestHandler

func TestMain(m *testing.M) {
	r := storage.NewDataHandler(&DefaultRedisDataTestConfig)
	r.Start()
	l, _ := zap.NewProduction()
	benchTestHandler = NewHandler(&DefaultHandlerTestConfig, r, l)
	err := r.Clear()
	log.Println(err)
	err = r.EnableZone(benchZone)
	log.Println(err)
	err = r.SetZoneConfigFromJson(benchZone, "{\"cname_flattening\": false}")
	log.Println(err)
	for _, cmd := range benchEntries {
		err := r.SetLocationFromJson(benchZone, cmd[0], cmd[1])
		if err != nil {
			log.Printf("[ERROR] 1: %s\n%s", err, cmd[1])
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

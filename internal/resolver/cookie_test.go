package resolver

import (
	"github.com/miekg/dns"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"testing"
	"z42-core/internal/storage"
	"z42-core/internal/test"
)

func TestCookie(t *testing.T) {
	RegisterTestingT(t)
	r := storage.NewDataHandler(&DefaultRedisDataTestConfig)
	r.Start()
	l, _ := zap.NewProduction()
	h := NewHandler(&DefaultHandlerTestConfig, r, l)
	err := h.RedisData.Clear()
	Expect(err).To(BeNil())
	err = r.EnableZone("example.com.")
	Expect(err).To(BeNil())
	err = r.SetLocationFromJson("example.com.", "@", `{"a":{"ttl":3600, "records":[{"ip":"1.2.3.4"}]}}`)
	Expect(err).To(BeNil())
	err = r.SetZoneConfigFromJson("example.com.", `{"soa":{"ttl":3600, "minttl":3600, "serial":1081539377, "mbox":"bugs.x.w.example.","ns":"ns1.example.com.","refresh":3600,"retry":300,"expire":3600000},"dnssec": false}`)
	Expect(err).To(BeNil())
	h.RedisData.LoadZones()

	m := new(dns.Msg)
	o := new(dns.OPT)
	o.Hdr.Name = "."
	o.Hdr.Rrtype = dns.TypeOPT
	e := new(dns.EDNS0_COOKIE)
	e.Code = dns.EDNS0COOKIE
	e.Cookie = "24a5ac1223344556"
	o.Option = append(o.Option, e)
	m.Extra = []dns.RR{o}

	w := test.NewRecorder(&test.ResponseWriter{})
	state := NewRequestContext(w, m)
	h.HandleRequest(state)

	resp := w.Msg
	Expect(resp.Rcode).To(Equal(dns.RcodeBadCookie))
}

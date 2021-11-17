package resolver

import (
	"errors"
	"fmt"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/internal/test"
	"github.com/hawell/z42/internal/types"
	"github.com/miekg/dns"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"sort"
	"strings"
	"testing"
)

var zone1ZskPriv = `
Private-key-format: v1.3
Algorithm: 5 (RSASHA1)
Modulus: oqwXm/EF8q6p5Rrj66Bbft+0Vk7Kj6TuvZp4nNl0htiT/8/92kIcri5gbxnV2v+p6jXYQI1Vx/vqP5cB0kPzjUQuJFVpm14fxOp89D6N0fPXR7xJ+SHs5nigHBIJdaP5
PublicExponent: AQAB
PrivateExponent: fJBa48aET3kAD7evn9aDOXwDk7Nx2NzrE7Uddr3tRPTDH7gdIuxNGfPZVDnsUG5EbX2JJf3JQsD7YXeQ+BGyytIi0ZTq8jsU63Np9hjheFx+IWSIz6S4JGnFKWRWUvuh
Prime1: 1c0EgZCXitPsdtEURwj1okEgzN9ze+QRP8adz0t+0s6ptB+bG3+YrhbzXcexE0tv
Prime2: wseiokM5ugXX0ZKy+8+lvumEZ94gM8Tyc031tFc1RRqIzB67k7139r/liNJoGXMX
Exponent1: WZyl79x3+CNdcGuv8RorQofDxLs/v0TXigCosnM1RAyFCs9Yhs0TZJyQAtWpPaoX
Exponent2: GXGcpBemBc/Xlm/UY6KHYz375tmUWU7j4P4RF6LAuasyrX9iP3Vjo18D6/CYWqK3
Coefficient: GhzOVUQcUJkvbYc9/+9MZngzDCeoetXDR6IILqG0/Rmt7FHWwSD7nOSoUUE5GslF
Created: 20180717134704
Publish: 20180717134704
Activate: 20180717134704
`

var zone1ZskPub = `
dnssec_test.com. IN DNSKEY 256 3 5 AwEAAaKsF5vxBfKuqeUa4+ugW37ftFZOyo+k7r2aeJzZdIbYk//P/dpC HK4uYG8Z1dr/qeo12ECNVcf76j+XAdJD841ELiRVaZteH8TqfPQ+jdHz 10e8Sfkh7OZ4oBwSCXWj+Q==
`

var zone1KskPriv = `
Private-key-format: v1.3
Algorithm: 5 (RSASHA1)
Modulus: 5WuOIP3GHID5Qmed6L+2ehBCkusTAXNv9uUfpzzTJHsA+bBesZSFsRNzMAV2drM7fApcL5IgNqrhb5twxu1/+cZj2Ld3PALbkENzn/erTl4A4uQdSWdkj8KnaLiJQPaT
PublicExponent: AQAB
PrivateExponent: BxiDhduzg/AtRXOE+8zqLO5R0M96gAH9BYripr6H3Un8prxgwWdRlz99wY95sYQrlNWr+4hhvikuOc9FjpXGg8E63iCNaZsVd/l8RvLGCtRPMtOEWhOecKe3kktHMUxp
Prime1: 9EWCZ3wwK2q7nsts12QuFGBTH/SOgHiaw9ieAn+mOA679BlIWXjeUoA5Hlj+ob31
Prime2: 8G9/lMOO+xgwjU7lQ5teFGmmNb2JXB/nP3pWQURdy+Chnb8wrcHALJGW1G7DAMVn
Exponent1: jroSoQ7iQmwh5n3sQcpqVkOWLmTB4vUVUPvAD6uwXq7VSaKAMK88EC6VsVLErZMF
Exponent2: qIlPwgTOzf3n0rXSCXD4IpDoHFWO2o/Wdm2X1spIgWglgcEKK1JcFiG7u48ki/7T
Coefficient: QCGY0yr+kkmOZfUoL9YCCgau/xjyEPRZgiGTfIy0PtGGMDKfUswJ+1KWI9Jue3E5
Created: 20190518113600
Publish: 20190518113600
Activate: 20190518113600
`

var zone1KskPub = `
dnssec_test.com. IN DNSKEY 257 3 5 AwEAAeVrjiD9xhyA+UJnnei/tnoQQpLrEwFzb/blH6c80yR7APmwXrGU hbETczAFdnazO3wKXC+SIDaq4W+bcMbtf/nGY9i3dzwC25BDc5/3q05e AOLkHUlnZI/Cp2i4iUD2kw==
`

var zone2ZskPub = `
example. IN DNSKEY 256 3 8 AwEAAZ4W6Du8WWtEPkvu51KL6RadzVfH4+T3LJFQKY4m2ftW9/Vg4Bkv KElxZY0GfUE2JkU3yqSKuEgEaNByjagU/RhwTm7ygsL5YF3b98Be6UGy PtZ0c4fefMdKMxBosIoRsNBVM31e+ruuxK9cG6kju6iTc63ADTIzVZwD j+vHVrYD
`

var zone2ZskPriv = `
Private-key-format: v1.3
Algorithm: 8 (RSASHA256)
Modulus: nhboO7xZa0Q+S+7nUovpFp3NV8fj5PcskVApjibZ+1b39WDgGS8oSXFljQZ9QTYmRTfKpIq4SARo0HKNqBT9GHBObvKCwvlgXdv3wF7pQbI+1nRzh958x0ozEGiwihGw0FUzfV76u67Er1wbqSO7qJNzrcANMjNVnAOP68dWtgM=
PublicExponent: AQAB
PrivateExponent: bxq0Tj86LNwCWEVnx6jSwPVYeofeT22zodDP07rUWgMuMwLJnIl669rJPwq/ftQ6o0zpmyhvCRYoP88yZV2S3b4JqU5R8DHHfBP3fFskLxyxscRr9nuk+l6EiywIaKhT+w6zMZGD7tn9nmdMCuAsZjVhCGa9mcpG6K2CvYBVvAk=
Prime1: 0mk7IP3c+gkcedOS/o7OGIEIGtbyMrqOeCH1LS9p9alaAakKkHqDHL+p968AhJHSWZsejGQ4e6mftrgL0p0xRQ==
Prime2: wFeUlblO74p9m7m88aIZdzihtl9B2gFF2JH9Mun2vI3nmK0t8YwhQpvMd4C9NStNUgLbj2EvKiuMq0/QEKFqpw==
Exponent1: Y0IJBrM7Pyh1KnNIcJVlW+HitOaZMp0XAEzkoAAx+BV/xDC+LxHcL/+qapE/qUow9NxcONY+XvfRxBxmV2CYEQ==
Exponent2: vmPzCGHt6N9FhqhMh0LVwlWkfUm9fXZVFRMtdwBw5CPzZAXIvJjhM3XU51Xf9IlweAWsIDkq3qtNCyZt5ohhcQ==
Coefficient: ZxO4iRgm90KiDRGE1C0ovPraJane9LtG7la4HbcAFZmVsTSpVkZJCBsqjwQP/hkqzizwWoZsU0KFsNG7Kugxjw==
Created: 20200204092511
Publish: 20200204092511
Activate: 20200204092511
`

var zone2KskPub = `
example. IN DNSKEY 257 3 8 AwEAAdgvqSHAbkXVALKEs2d9XE88ezp6zNj049xiUhZ25JtyNaCrE8qX o8vaJPCZ+uHDNbO8kqIGe3BCUG9NiC/2ryJThyZnEt8hEdm2p0zyKHci 2IJq7U2XscczYFDnkkzQj9hiGWncBrMRtpakNyLFLNV0ZL18os7QFjfa SY9iaREnS9TLYmn3bsbNg1oBSbKtBhW0RAdzwXW2zWT6gAhfx5M/Tbot 6RWmyRrL8QbjagZiISQyk6jCabppMGLzEp/pqFR9z8GmF5WrKTJtmd0P Gq7GGiPrK5yDh2NEF2WLr5/QqLdgJNOaLuI+7RiBkg4UaVyz+Nca6B2k CzZgAXYxHas=
`

var zone2KskPriv = `
Private-key-format: v1.3
Algorithm: 8 (RSASHA256)
Modulus: 2C+pIcBuRdUAsoSzZ31cTzx7OnrM2PTj3GJSFnbkm3I1oKsTypejy9ok8Jn64cM1s7ySogZ7cEJQb02IL/avIlOHJmcS3yER2banTPIodyLYgmrtTZexxzNgUOeSTNCP2GIZadwGsxG2lqQ3IsUs1XRkvXyiztAWN9pJj2JpESdL1Mtiafduxs2DWgFJsq0GFbREB3PBdbbNZPqACF/Hkz9Nui3pFabJGsvxBuNqBmIhJDKTqMJpumkwYvMSn+moVH3PwaYXlaspMm2Z3Q8arsYaI+srnIOHY0QXZYuvn9Cot2Ak05ou4j7tGIGSDhRpXLP41xroHaQLNmABdjEdqw==
PublicExponent: AQAB
PrivateExponent: bSnb6LwnssF1Aa/6e4aUxzoOK6B4shEuwkkvlEJi+493PvNEIifiQPydbJUEV13gTysojAJj8HK79QgcfcO9+cJd22lu4Rbs0Zfm8PbSsh35YBmoTGcOET2DJDda68jg6e3XUVoWU/Pc1EKFyNvx4LNOb1RxTadLoNZsEKgrz8mvKQF/xtoHFirLfVEiCywWD0hcVR3FqqwibX6eYnMl2NzhBfzKVKIWm94gNf06MAOX2aAFE2NvMkrILJuv8ML9VPo86ROUuqCrCok0tVN6/z7tHWj6tKsxtuXftjYBGgXIUKGZ0tIBDaRNZQ/3wsm8ECPO/Nb4OhMfbBE1lj+SAQ==
Prime1: 9ru3QOSUrYvQcY/pfY1keo5Ugjqqb+rWKnXMJpbLm8rVROIdHiK7g85OHyzruDqmv1l1Q5lpp2/z9yEGfp6tO/CLlqbG0MKPP5zESI2OlqVkcoMAEtdMasXQg1nBjW8VWky+xaneI5sj/KkvXsDd4XXk/BIRVVwDrsJuGXxQYQE=
Prime2: 4E491BVNur0U/bzIWSamz7eAtuOJWGiOLhiVIrRtX7KUBnRmOWWjJ+stb8q3ilVyJ/0RyC+GCiaMcV6r3ThwZjulaF49Pp+EHasCnhWBZShVOHp2jHYsAgq86p3P8NIH/Qpr0jjAQtLHlgA1mElLfPzFeqgbHhPLCRswAP9uUqs=
Exponent1: gDmgC/Z/Gg3+PvZmhtxTaqnLW363ksA9mwVrGmbl28o2uby1GzM7tk0iJmuG+VBp1incmkwBL4YsCLO+F1HJf8wMDzgPPPDP12RWUcpXXw0HPce84w3G5fp12b1srF8dfrdBsaINEv4OXsFiH+ElroVBgoq1PWI7e7gJ1e7YKwE=
Exponent2: PQHq3SFCN/UvnWfYUi8qFbsCXjv64jnl2fHDtmG+kdW/XxYPq7LSMoxLmmlXjF97Ihc52+nZGi+r6TXnps6v+45jicSAAeVfCLa3ioms3PegXjEox0Fo7NFA2ss7gHOPyqon81COMl6j/E9oRFhDGOajS54nagHWKk7jupG+zus=
Coefficient: XnSHdesZGz3tYIztsIrS/ZDHnXTtRC0Ags9r1QZCq95lvTq2AvB2XQbdHml+aauJUGrHrK90zJq9omquknvX/PgFoAR09XkL3TkPLaRcJG8v2tjtI5hkqBaCp1BN2UmcTQsPoLn9IGorph0rU6GQ9FAqT2OXC5JIYwVhDpI5Vsg=
Created: 20200204092439
Publish: 20200204092439
Activate: 20200204092439
`

func DefaultDnssecInitialize(zskPub, zskPriv, kskPub, kskPriv string) func(testCase *TestCase) (requestHandler *DnsRequestHandler, e error) {
	return func(testCase *TestCase) (requestHandler *DnsRequestHandler, e error) {
		r := storage.NewDataHandler(&DefaultRedisDataTestConfig)
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
					return nil, errors.New(fmt.Sprintf("[ERROR] 2: %s\n%s", err, cmd[1]))
				}
			}
			if err := r.SetZoneConfigFromJson(zone, testCase.ZoneConfigs[i]); err != nil {
				return nil, err
			}
			if err := h.RedisData.SetZoneKey(zone, "zsk", zskPub, zskPriv); err != nil {
				zap.L().Error("cannot set zsk", zap.Error(err))
			}
			if err := h.RedisData.SetZoneKey(zone, "ksk", kskPub, kskPriv); err != nil {
				zap.L().Error("cannot set ksk", zap.Error(err))
			}
		}
		h.RedisData.LoadZones()
		return h, nil
	}
}

func DefaultDnssecApplyAndVerify(testCase *TestCase, requestHandler *DnsRequestHandler, t *testing.T) {
	RegisterTestingT(t)
	var zsk dns.RR
	var ksk dns.RR
	{
		dnskeyQuery := test.Case{
			Do:    true,
			Qname: testCase.Zones[0], Qtype: dns.TypeDNSKEY,
		}
		r := dnskeyQuery.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		requestHandler.HandleRequest(state)
		resp := w.Msg
		// fmt.Println(resp.Answer)
		for _, answer := range resp.Answer {
			if key, ok := answer.(*dns.DNSKEY); ok {
				if key.Flags == 256 {
					zsk = answer
				} else if key.Flags == 257 {
					ksk = answer
				}
			}
		}
	}
	// fmt.Println("zsk is ", zsk.String())
	// fmt.Println("ksk is ", ksk.String())

	for i, tc0 := range testCase.TestCases {
		// fmt.Println(i)
		tc := test.Case{
			Desc:  testCase.TestCases[i].Desc,
			Qname: testCase.TestCases[i].Qname, Qtype: testCase.TestCases[i].Qtype,
			Answer: make([]dns.RR, len(testCase.TestCases[i].Answer)),
			Ns:     make([]dns.RR, len(testCase.TestCases[i].Ns)),
			Do:     testCase.TestCases[i].Do,
			Extra:  make([]dns.RR, len(testCase.TestCases[i].Extra)),
		}
		if !tc.Do {
			tc.Extra = []dns.RR{}
		}
		copy(tc.Answer, testCase.TestCases[i].Answer)
		copy(tc.Ns, testCase.TestCases[i].Ns)
		copy(tc.Extra, testCase.TestCases[i].Extra)
		sort.Sort(test.RRSet(tc.Answer))
		sort.Sort(test.RRSet(tc.Ns))
		sort.Sort(test.RRSet(tc.Extra))

		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		requestHandler.HandleRequest(state)
		resp := w.Msg
		for _, rrs := range [][]dns.RR{tc0.Answer, tc0.Ns, resp.Answer, resp.Ns} {
			sets := types.SplitSets(rrs)
			rrsigs := make(map[types.RRSetKey]*dns.RRSIG)
			for _, rr := range rrs {
				if rrsig, ok := rr.(*dns.RRSIG); ok {
					rrsigs[types.RRSetKey{QName: rrsig.Hdr.Name, QType: rrsig.TypeCovered}] = rrsig
				}
			}
			for _, set := range sets {
				rrsig := rrsigs[types.RRSetKey{QName: set[0].Header().Name, QType: set[0].Header().Rrtype}]
				if rrsig == nil {
					continue
				}
				// FIXME: should it be set[0].Header().Rrtype?
				if tc.Qtype == dns.TypeDNSKEY {
					err := rrsig.Verify(ksk.(*dns.DNSKEY), set)
					Expect(err).To(BeNil())
				} else {
					err := rrsig.Verify(zsk.(*dns.DNSKEY), set)
					Expect(err).To(BeNil())
				}
			}
		}
		//fmt.Println("dddd")
		err := test.SortAndCheck(resp, tc)
		Expect(err).To(BeNil())
		//fmt.Println("xxxx")
	}
}

var dnssecTestCases = []*TestCase{
	{
		Name:           "dnssec test",
		Description:    "test basic dnssec functionality",
		Enabled:        true,
		HandlerConfig:  DefaultHandlerTestConfig,
		Initialize:     DefaultDnssecInitialize(zone1ZskPub, zone1ZskPriv, zone1KskPub, zone1KskPriv),
		ApplyAndVerify: DefaultDnssecApplyAndVerify,
		Zones:          []string{"dnssec_test.com."},
		ZoneConfigs:    []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.dnssec_test.com.","ns":"ns1.dnssec_test.com.","refresh":44,"retry":55,"expire":66},"dnssec": true}`},
		Entries: [][][]string{
			{
				{"@",
					`{"ns":{"ttl":300,"records":[{"host":"ns1.dnssec_test.com."},{"host":"ns2.dnssec_test.com."}]}}`,
				},
				{"x",
					`{
			            "a":{"ttl":300, "records":[{"ip":"1.2.3.4", "country":["ES"]},{"ip":"5.6.7.8", "country":[""]}]},
            			"aaaa":{"ttl":300, "records":[{"ip":"::1"}]},
            			"txt":{"ttl":300, "records":[{"text":"foo"},{"text":"bar"}]},
            			"mx":{"ttl":300, "records":[{"host":"mx1.dnssec_test.com.", "preference":10},{"host":"mx2.dnssec_test.com.", "preference":10}]},
            			"srv":{"ttl":300, "records":[{"target":"sip.dnssec_test.com.","port":555,"priority":10,"weight":100}]}
        			}`,
				},
				{"y",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.dnssec_test.com."},{"host":"ns2.dnssec_test.com."}]}}`,
				},
				{"*",
					`{"txt":{"ttl":300,"records":[{"text":"wildcard text"}]}}`,
				},
				{"a",
					`{"a":{"ttl":300,"records":[{"ip":"129.0.2.1"}]},"txt":{"ttl":300,"records":[{"text":"a text"}]}}`,
				},
				{"d",
					`{"a":{"ttl":300,"records":[{"ip":"129.0.2.1"}]},"txt":{"ttl":300,"records":[{"text":"d text"}]}}`,
				},
				{"c1",
					`{"cname":{"ttl":300, "host":"c2.dnssec_test.com."}}`,
				},
				{"c2",
					`{"cname":{"ttl":300, "host":"c3.dnssec_test.com."}}`,
				},
				{"c3",
					`{"cname":{"ttl":300, "host":"a.dnssec_test.com."}}`,
				},
				{"w",
					`{"cname":{"ttl":300, "host":"w.a.dnssec_test.com."}}`,
				},
				{"*.a",
					`{"cname":{"ttl":300, "host":"w.b.dnssec_test.com."}}`,
				},
				{"*.b",
					`{"cname":{"ttl":300, "host":"w.c.dnssec_test.com."}}`,
				},
				{"*.c",
					`{"a":{"ttl":300, "records":[{"ip":"129.0.2.1"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "DNSKEY test",
				Qname: "dnssec_test.com.", Qtype: dns.TypeDNSKEY,
				Answer: []dns.RR{
					test.DNSKEY("dnssec_test.com.	3600	IN	DNSKEY	256 3 5 AwEAAaKsF5vxBfKuqeUa4+ugW37ftFZOyo+k7r2aeJzZdIbYk//P/dpCHK4uYG8Z1dr/qeo12ECNVcf76j+XAdJD841ELiRVaZteH8TqfPQ+jdHz10e8Sfkh7OZ4oBwSCXWj+Q=="),
					test.DNSKEY("dnssec_test.com.	3600	IN	DNSKEY	257 3 5 AwEAAeVrjiD9xhyA+UJnnei/tnoQQpLrEwFzb/blH6c80yR7APmwXrGUhbETczAFdnazO3wKXC+SIDaq4W+bcMbtf/nGY9i3dzwC25BDc5/3q05eAOLkHUlnZI/Cp2i4iUD2kw=="),
					test.RRSIG("dnssec_test.com.	3600	IN	RRSIG	DNSKEY 5 2 3600 20190527081109 20190519051109 37456 dnssec_test.com. oVwtVEf9eOkcuSJlsH0OSBUvLOxgKM1pIAe7v717oRyCoyC+FIG5uGsdrZWhgklh/fpEmRdJQ+nHXKWT/son8zvxAoskuIIp49wwgvcS400IoHiyjIY0BHNTFPvsPdy0"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			{
				Desc:  "DO=false test",
				Qname: "x.dnssec_test.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("x.dnssec_test.com. 300 IN A 1.2.3.4"),
					test.A("x.dnssec_test.com. 300 IN A 5.6.7.8"),
				},
			},
			{
				Desc:  "A test",
				Qname: "x.dnssec_test.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("x.dnssec_test.com. 300 IN A 1.2.3.4"),
					test.A("x.dnssec_test.com. 300 IN A 5.6.7.8"),
					test.RRSIG("x.dnssec_test.com.	300	IN	RRSIG	A 5 3 300 20180726080503 20180718050503 22548 dnssec_test.com. b/rdGOnMQzKX4K9c3CLvJYb2ErFlrShy8vBh86Y28t1RRnN9OCj7L1AGhr+5xEge3mpuRNd2djXFh7CwZmAOm6R0/acRP1mw1RnlSANhaVt1Enr57c6+5grPgn7e45X3"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			{
				Desc:  "AAAA test",
				Qname: "x.dnssec_test.com.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("x.dnssec_test.com. 300 IN AAAA ::1"),
					test.RRSIG("x.dnssec_test.com.	300	IN	RRSIG	AAAA 5 3 300 20180726102716 20180718072716 22548 dnssec_test.com. Bl6GjbEY2jXyWhVuQzehQs4RVvrIRvLz72eXjvRKXTg6BGmcZF7CyZo1+R2w3p83gAA0yhs6UnSD/GMC5zmLeR5/8LiTzWa0S5f5xZNHwWNEUtrtnS7nGCCFDXfLUI3n"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// TXT Test
			{
				Desc:  "TXT test",
				Qname: "x.dnssec_test.com.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("x.dnssec_test.com. 300 IN TXT bar"),
					test.TXT("x.dnssec_test.com. 300 IN TXT foo"),
					test.RRSIG("x.dnssec_test.com.	300	IN	RRSIG	TXT 5 3 300 20180726102908 20180718072908 22548 dnssec_test.com. NND6mWXgQ1CY/KTsgPcjvty7FdLCFQdoHQ6Rmyv2hpPg12xTmAokB/TScTeL+zhvtt+9ktYnErspZc3LVoyPqZ8TYppHHoEXDR8OpyqmVcTPx/fzRuW5zmuUpofnhlV6"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// NS Test
			{
				Desc:  "NS test",
				Qname: "dnssec_test.com.", Qtype: dns.TypeNS,
				Answer: []dns.RR{
					test.NS("dnssec_test.com. 300 IN NS ns1.dnssec_test.com."),
					test.NS("dnssec_test.com. 300 IN NS ns2.dnssec_test.com."),
					test.RRSIG("dnssec_test.com.	300	IN	RRSIG	NS 5 2 300 20191122140218 20191114110218 22548 dnssec_test.com. DK9giOXPadNyDfFtsPjEd9JpWGIeCyIOgDDwzvgsYc/k/Q5blgtWBNxJ Fk0aPqj6M15RFTig2nA3uEJpEJx7OAj5zzSSTqyPozT/qrmPMWdxJcuK yaJ+CH+Ws9wJsM3S"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// MX Test
			{
				Desc:  "MX test",
				Qname: "x.dnssec_test.com.", Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("x.dnssec_test.com. 300 IN MX 10 mx1.dnssec_test.com."),
					test.MX("x.dnssec_test.com. 300 IN MX 10 mx2.dnssec_test.com."),
					test.RRSIG("x.dnssec_test.com.	300	IN	RRSIG	MX 5 3 300 20180726104823 20180718074823 22548 dnssec_test.com. I0il28K7OmjA/hRwV/uPyieeg+EnpxRQmcUvZ1JsijIAqf6FVqDbysgrZfzZBheizMuLsEjPmmVTJrl34Y1ZEHxwD9oxgxWSDQ4L7kHLUeOSTRA73maHOtr+Sypygw6E"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// SRV Test
			{
				Desc:  "SRV test",
				Qname: "x.dnssec_test.com.", Qtype: dns.TypeSRV,
				Answer: []dns.RR{
					test.SRV("x.dnssec_test.com. 300 IN SRV 10 100 555 sip.dnssec_test.com."),
					test.RRSIG("x.dnssec_test.com.	300	IN	RRSIG	SRV 5 3 300 20180726104916 20180718074916 22548 dnssec_test.com. hwyeNmMQ6K6Ja/ogepGQvGEyEiBeCd7Suhb6CL/uEREuREq1wcr9QhS2s3yKy9ZhjO9xs2x38vSSZHvRBvTjVxMIpPaQuxcWI02s/NgVLkRA5H0LpBPE5pyXDxTmtavV"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// SOA Test
			{
				Desc:  "SOA test",
				Qname: "dnssec_test.com.", Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("dnssec_test.com.	300	IN	SOA	ns1.dnssec_test.com. hostmaster.dnssec_test.com. 1533107401 44 55 66 100"),
					test.RRSIG("dnssec_test.com.	300	IN	RRSIG	SOA 5 2 300 20180809071001 20180801041001 22548 dnssec_test.com. O4+6kPz9sr26RDZLy9MUoQRFweEzVZJ8JvQAJ+3mcZ/xO8z4KKNRb3Gpf7sWyoQk6Bd476VkZHbkbEf9SRptDqDHPV5MxMDUa3AtbdwUkRaVDidL95B4KDcno5FOU55I"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// NXDomain Test
			{
				Desc:  "NXDOMAIN test",
				Qname: "nxdomain.x.dnssec_test.com.", Qtype: dns.TypeAAAA,
				Ns: []dns.RR{
					test.SOA("dnssec_test.com.	300	IN	SOA	ns1.dnssec_test.com. hostmaster.dnssec_test.com. 1533107621 44 55 66 100"),
					test.RRSIG("dnssec_test.com.	300	IN	RRSIG	SOA 5 2 300 20180809071341 20180801041341 22548 dnssec_test.com. hJ6GxQo46z5hxBV48hs5Ab1tdfCJ1S7wxIIoI3cksCtf+dqv/eLmlxGH0KuEabAPWhp9VqyjjQYxvSP/0gH0Z/BwYxoghxrROuqHqiIbkbM8wvgLHBwNv+vA4xXUN/Ej"),
					test.NSEC("nxdomain.x.dnssec_test.com.	100	IN	NSEC	\\000.nxdomain.x.dnssec_test.com. RRSIG NSEC"),
					test.RRSIG("nxdomain.x.dnssec_test.com.	100	IN	RRSIG	NSEC 5 4 100 20180809115341 20180801085341 22548 dnssec_test.com. cHqIhWUalUAib9cpVd+4XLLzxrm6zKiQKLWs1/2T4dNhaS/CAkIXY6so0YDpsm0wgS2McpVd/GL+2fPDEb0MXJYyTfX8mzn5i49riQjEiHbmlL7oZfXCUKxKTRYczxjf"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// wildcard Test
			{
				Desc:  "wildcard test",
				Qname: "z.dnssec_test.com.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("z.dnssec_test.com. 300 IN TXT \"wildcard text\""),
					test.RRSIG("z.dnssec_test.com.	300	IN	RRSIG	TXT 5 3 300 20180731095235 20180723065235 22548 dnssec_test.com. YCmkNMLkg6qtey+9+Yt+Jq0V1itDF9Gw8rodPk82b486jE22xxleLq8zcwne8Xekp57H/9Sk5mmTzczWTZQAUauUQF+o2QzLkgiI5vr0gtC5Y3fraRCDclo9/8IQ2yEs"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// cname flattening test
			{
				Desc:  "cname flattening test",
				Qname: "c1.dnssec_test.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("c1.dnssec_test.com.	300	IN	A	129.0.2.1"),
					test.RRSIG("c1.dnssec_test.com.	300	IN	RRSIG	A 5 3 300 20200210125610 20200202095610 22548 dnssec_test.com. jlann44kdNt5V84SxQRGTA3OjU6/SEZ/eakCu39C9FrtkaLyXxsmkO5cMqJ9XuO5WY2a6TJjtiF3JbM/daVJIfnL3Qx9IckAXt/dzJBr3ymM2dRvomhtMXtbs5ftfPXa"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// CNAME flattening + wildcard Test
			{
				Desc:  "cname flattening + wildcard",
				Qname: "w.dnssec_test.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("w.dnssec_test.com.	300	IN	A	129.0.2.1"),
					test.RRSIG("w.dnssec_test.com.	300	IN	RRSIG	A 5 3 300 20200210125029 20200202095029 22548 dnssec_test.com. Cwan6/FCm94xVeZBI9VYRLtRfW6z8/hyRjTaK53HzuluFkR61/hVvOteS0daLHiusPoLppJGWleB8LeNSA7r0frQJXpcHOQyhRiV2M7nIo2MFf+H+WPZeyLCqbV+0XK/"),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			// NSEC at zone cut
			// NSEC at root
			// NSEC at cname
			// NSEC with empty response
			// NSEC with nxdomain
			// NSEC with wildcard
			// DS at root
			// DS at zone cut
		},
	},
	{
		Name:           "RFC4035",
		Description:    "RFC4035 sample zone",
		Enabled:        true,
		HandlerConfig:  DefaultHandlerTestConfig,
		Initialize:     DefaultDnssecInitialize(zone2ZskPub, zone2ZskPriv, zone2KskPub, zone2KskPriv),
		ApplyAndVerify: DefaultDnssecApplyAndVerify,
		Zones:          []string{"example."},
		ZoneConfigs:    []string{`{"soa":{"ttl":3600, "minttl":3600, "serial":1081539377, "mbox":"bugs.x.w.example.","ns":"ns1.example.","refresh":3600,"retry":300,"expire":3600000},"dnssec": true}`},
		Entries: [][][]string{
			{
				{"@",
					`{
						"ns":{"ttl":3600, "records":[{"host":"ns1.example."},{"host":"ns2.example."}]},
						"mx":{"ttl":3600, "records":[{"host":"xx.example.", "preference":1}]}
					}`,
				},
				{"a",
					`{
						"ns":{"ttl":3600, "records":[{"host":"ns1.a.example."},{"host":"ns2.a.example."}]},
						"ds":{"ttl":3600, "records":[{"key_tag":57855, "algorithm":5, "digest_type":1, "digest":"B6DCD485719ADCA18E5F3D48A2331627FDD3636B"}]}
					}`,
				},
				{"ns1.a",
					`{"a":{"ttl":3600, "records":[{"ip":"192.0.2.5"}]}}`,
				},
				{"ns2.a",
					`{"a":{"ttl":3600, "records":[{"ip":"192.0.2.6"}]}}`,
				},
				{"ai",
					`{
						"a":{"ttl":3600, "records":[{"ip":"192.0.2.9"}]},
						"hinfo":{"cpu":"KLH-10", "os":"ITS"},
						"aaaa":{"ttl":3600, "records":[{"ip":"2001:db8::f00:baa9"}]}
					}`,
				},
				{"b",
					`{
						"ns":{"ttl":3600, "records":[{"host":"ns1.b.example."},{"host":"ns2.b.example."}]}
					}`,
				},
				{"ns1.b",
					`{"a":{"ttl":3600, "records":[{"ip":"192.0.2.7"}]}}`,
				},
				{"ns2.b",
					`{"a":{"ttl":3600, "records":[{"ip":"192.0.2.8"}]}}`,
				},
				{"ns1",
					`{"a":{"ttl":3600, "records":[{"ip":"192.0.2.1"}]}}`,
				},
				{"ns2",
					`{"a":{"ttl":3600, "records":[{"ip":"192.0.2.1"}]}}`,
				},
				{"*.w",
					`{"mx":{"ttl":3600, "records":[{"host":"ai.example.", "preference":1}]}}`,
				},
				{"x.w",
					`{"mx":{"ttl":3600, "records":[{"host":"xx.example.", "preference":1}]}}`,
				},
				{"x.y.w",
					`{"mx":{"ttl":3600, "records":[{"host":"xx.example.", "preference":1}]}}`,
				},
				{"xx",
					`{
						"a":{"ttl":3600, "records":[{"ip":"192.0.2.10"}]},
						"hinfo":{"cpu":"KLH-10", "os":"TOPS-20"},
						"aaaa":{"ttl":3600, "records":[{"ip":"2001:db8::f00:baaa"}]}
					}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "A successful query to an authoritative server",
				Qname: "x.w.example.",
				Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("x.w.example.   3600 IN MX  1 xx.example."),
					test.RRSIG("x.w.example.	3600	IN	RRSIG	MX 8 3 3600 20200216150627 20200208120627 28743 example. OxWtte5eF1MIeQkeXL8V/aF06wA30YbH/QRue/FS/JqJFFWTqs+vzPtyz14csJgJfk1aaCMVJmdWcqZQd0rHj2HedTc7nmP9RlNHYRLYXVGAyOa2WMew55OnrUVhOTmfBeIvIHCIECGi7DhT+Bkr6SvlC87ah9T09TZ47v5DcU0="),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			{
				Desc:  "An authoritative name error",
				Qname: "ml.example.",
				Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.SOA("example. 3600 IN SOA ns1.example. bugs.x.w.example. 1081539377 3600 300 3600000 3600"),
					test.RRSIG("example.	3600	IN	RRSIG	SOA 8 1 3600 20200216151051 20200208121051 28743 example. dSsS8DyvgC5ZJR38JDPmMYeZaQYSC6vjsyAz3XQgTIa3KTqHPPgtAF7uIAPNuYW8j5pb6WNUysNfg2lFiefY68VZu5oFYNONlqU956jFinK71nYF8btSOruWHPlmLhiu0pbsHZI9EefObTEnET9wsC+YGQnSKADIkyjy2Chljnk="),
					test.NSEC("ml.example. 3600 IN NSEC \\000.ml.example. RRSIG NSEC"),
					test.RRSIG("ml.example.	3600	IN	RRSIG	NSEC 8 2 3600 20200216151051 20200208121051 28743 example. jaNMf4h1d9HHHF+e0aPN/j93LIiEPdnQFhQ9GhjLFPf2te1dwPXh1nHpLxvlzpyG31lSmPze8ktcZSngAr8SpP0IDfEzhV77JR6QmRJh54g9JWExgRnMRY+Wr+lp+Vv9rdfuPpQOkGf2WZsPGuq7IYtXSCIMhnXngU03irEthW8="),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			{
				Desc:  "A NODATA response. The NSEC RR proves that the name exists and that the requested RR type does not",
				Qname: "ns1.example.",
				Qtype: dns.TypeMX,
				Ns: []dns.RR{
					test.SOA("example. 3600 IN SOA ns1.example. bugs.x.w.example. 1081539377 3600 300 3600000 3600"),
					test.RRSIG("example.	3600	IN	RRSIG	SOA 8 1 3600 20200216151051 20200208121051 28743 example. dSsS8DyvgC5ZJR38JDPmMYeZaQYSC6vjsyAz3XQgTIa3KTqHPPgtAF7uIAPNuYW8j5pb6WNUysNfg2lFiefY68VZu5oFYNONlqU956jFinK71nYF8btSOruWHPlmLhiu0pbsHZI9EefObTEnET9wsC+YGQnSKADIkyjy2Chljnk="),
					test.NSEC("ns1.example. 3600 IN NSEC \\000.ns1.example. A CNAME PTR TXT AAAA SRV RRSIG NSEC TLSA CAA"),
					test.RRSIG("ns1.example.	3600	IN	RRSIG	NSEC 8 2 3600 20200212122320 20200204092320 28743 example. PXRFcyuDlGx4t7SXBPOL6M0r1lxlectaATAoqruOhWcJcmOKgVvuV0A+RccLuNwiQzqa2CV2np2ZXXt1F42fHd+WtnpYeT5btZEeICGRgt1S0ABUuIlhARmow0wxEeFmQUrteEDKfVsOuDrGL0sAxCe8c1zq1acwntVz1VqtLzM="),
				},
				Do: true,
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
			},
			{
				Desc:  "Referral to a signed zone",
				Qname: "mc.a.example.",
				Qtype: dns.TypeMX,
				Ns: []dns.RR{
					test.NS("a.example. 3600 IN NS ns1.a.example."),
					test.NS("a.example. 3600 IN NS ns2.a.example."),
					test.DS("a.example. 3600 DS 57855 5 1 B6DCD485719ADCA18E5F3D48A2331627FDD3636B"),
					test.RRSIG("a.example.	3600	IN	RRSIG	DS 8 2 3600 20200217143703 20200209113703 28743 example. Q/17Xktimq5Evf1WO3mRyY1X8smGug2qUNPgSQIPwh+maQdOChfSRQjWA86RzF5VgudaCriCN3RUoae3CzA9c+MBS6wNU+7aaLsa9klBqxsWE2luK+b3xf8RtBXQJfRCisrllQl2qJRkavafPyXlVhHvm4m82BkA8tiM1cUdoqM="),
				},
				Extra: []dns.RR{
					test.A("ns1.a.example. 3600 IN A   192.0.2.5"),
					test.A("ns2.a.example. 3600 IN A   192.0.2.6"),
					test.OPT(4096, true),
				},
				Do: true,
			},
			{
				Desc:  "Referral to an unsigned zone",
				Qname: "mc.b.example.",
				Qtype: dns.TypeMX,
				Ns: []dns.RR{
					test.NS("b.example. 3600 IN NS ns1.b.example."),
					test.NS("b.example. 3600 IN NS ns2.b.example."),
					test.NSEC("b.example. 3600 NSEC \\000.b.example. NS RRSIG NSEC"),
					test.RRSIG("b.example.	3600	IN	RRSIG	NSEC 8 2 3600 20200218121138 20200210091138 28743 example. hcwzQKkjQXa1zTR7x2PxFxpxQLEbszw7gO2DrC8SpSRpjtcC+rs+oPXFm8vgmlD6uf5J4/pkIwA6iWxuuwMLIicQH0aq9nr/XqQrgOasVzm4sc4K0KJWdupI0g9G60uPFua1m4SXYtx3/2RsXfvjuTVj/uoG5eN/z69ABTcwFKY="),
				},
				Extra: []dns.RR{
					test.A("ns1.b.example. 3600 IN A   192.0.2.7"),
					test.A("ns2.b.example. 3600 IN A   192.0.2.8"),
					test.OPT(4096, true),
				},
				Do: true,
			},
			{
				Desc:  "A successful query that was answered via wildcard expansion",
				Qname: "a.z.w.example.",
				Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("a.z.w.example. 3600 IN MX  1 ai.example."),
					test.RRSIG("a.z.w.example.	3600	IN	RRSIG	MX 8 4 3600 20200218121716 20200210091716 28743 example. BVb7G5aro3NpEn9MuQUryottex4/QiddmWRHX9xcGMvsdXxSh4uPqe32Z//yxNakJXUT6/1rbJ4gj7eJDrbqw03tdphVWfj26Frwm2hBKvN+hpWaBaOG5SQ1Yslw/LTI0SYEKcz2D8jdMArzWf8JlrjLmXdZYhRVcthpjgYu5w8="),
				},
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
				Do: true,
			},
			{
				Desc:  "A NODATA response for a name covered by a wildcard",
				Qname: "a.z.w.example.",
				Qtype: dns.TypeAAAA,
				Ns: []dns.RR{
					test.SOA("example. 3600 IN SOA ns1.example. bugs.x.w.example. 1081539377 3600 300 3600000 3600"),
					test.RRSIG("example.	3600	IN	RRSIG	SOA 8 1 3600 20200218121950 20200210091950 28743 example. lqLI5JNgfRJ/11i2La2Ydk1qeW6wyv5Acbxxwleq0Rxp8H0d0yuaarjII6IlRmI8SSq0tuEFFqiXhRp5Fn5jAmR6zMB9XtSFVG1yv3NrcmUurVx2wuuvjI3Oft2+UFq74gmDM0oOFxZHpWy0Zur90/KOlANjd/tfBnCNSQydinA="),
					test.NSEC("a.z.w.example. 3600 IN NSEC \\000.a.z.w.example. A CNAME PTR MX TXT SRV RRSIG NSEC TLSA CAA"),
					test.RRSIG("a.z.w.example.	3600	IN	RRSIG	NSEC 8 4 3600 20200218235854 20200210205854 28743 example. K7fb20RzhxAlosOmIuolGeuPxIUuY5fTHwtzQoPmLRF3iTH23ICjvWBVX4mx8aEAGlwXIik68rx7WCmuSG0+RbNDbRVfB540q5PT2Y0QoG4gTzueOUR0eboU7qxyAJ+w6KjiNzM0TQ8+SWkVT83gryff2jZAQDJQrMCMxAVV0JI="),
				},
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
				Do: true,
			},
			{
				Desc:  "QTYPE=DS query that was mistakenly sent to a name server for the child zone",
				Qname: "example.",
				Qtype: dns.TypeDS,
				Ns: []dns.RR{
					test.SOA("example. 3600 IN SOA ns1.example. bugs.x.w.example. 1081539377 3600 300 3600000 3600"),
					test.RRSIG("example.	3600	IN	RRSIG	SOA 8 1 3600 20200218121950 20200210091950 28743 example. lqLI5JNgfRJ/11i2La2Ydk1qeW6wyv5Acbxxwleq0Rxp8H0d0yuaarjII6IlRmI8SSq0tuEFFqiXhRp5Fn5jAmR6zMB9XtSFVG1yv3NrcmUurVx2wuuvjI3Oft2+UFq74gmDM0oOFxZHpWy0Zur90/KOlANjd/tfBnCNSQydinA="),
					test.NSEC("example.	3600	IN	NSEC	\\000.example. A NS SOA PTR MX TXT AAAA SRV RRSIG NSEC TLSA CAA"),
					test.RRSIG("example.	3600	IN	RRSIG	NSEC 8 1 3600 20200219000030 20200210210030 28743 example. hey30Zt0MMAuvYKhETYD6RXAK2guvQo72n4kANyOza0EP6Zcw0cp8ol6ZOpex8MavyuGmFXp4FPCVM9H682MIYfXNDCGPu/MFTJrhVaWdK/G9ng0/1ywgj9EYfbIYYlhrGb7g94aP95rX8X/O1zH/+MpRIIj0ZGUNm8YJkzI02U="),
				},
				Extra: []dns.RR{
					test.OPT(4096, true),
				},
				Do: true,
			},
		},
	},
}

func TestAllDnssec(t *testing.T) {
	RegisterTestingT(t)
	for _, testCase := range dnssecTestCases {
		if !testCase.Enabled {
			continue
		}
		fmt.Println(">>> ", CenterText(testCase.Name, 70), " <<<")
		fmt.Println(testCase.Description)
		fmt.Println(strings.Repeat("-", 80))
		h, err := testCase.Initialize(testCase)
		Expect(err).To(BeNil())
		testCase.ApplyAndVerify(testCase, h, t)
		fmt.Println(strings.Repeat("-", 80))
	}
}

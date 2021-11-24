package resolver

import (
	"errors"
	"fmt"
	"github.com/hawell/z42/internal/dnssec"
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

func DefaultDnssecInitialize() func(testCase *TestCase) (requestHandler *DnsRequestHandler, e error) {
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
			keys, _ := dnssec.GenerateKeys(zone)
			if err := h.RedisData.SetZoneKey(zone, "zsk", keys.ZSKPublic, keys.ZSKPrivate); err != nil {
				zap.L().Error("cannot set zsk", zap.Error(err))
			}
			if err := h.RedisData.SetZoneKey(zone, "ksk", keys.KSKPublic, keys.KSKPrivate); err != nil {
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
		var rrsig *dns.RRSIG
		for _, answer := range resp.Answer {
			if key, ok := answer.(*dns.DNSKEY); ok {
				if key.Flags == 256 {
					zsk = answer
				} else if key.Flags == 257 {
					ksk = answer
				}
			} else if rr, ok := answer.(*dns.RRSIG); ok {
				rrsig = rr
			}
		}
		Expect(ksk).NotTo(BeNil())
		Expect(zsk).NotTo(BeNil())
		Expect(rrsig).NotTo(BeNil())
		err := rrsig.Verify(ksk.(*dns.DNSKEY), []dns.RR{ksk, zsk})
		Expect(err).To(BeNil())
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

		r := tc.Msg()
		w := test.NewRecorder(&test.ResponseWriter{})
		state := NewRequestContext(w, r)
		requestHandler.HandleRequest(state)
		resp := w.Msg
		for _, section := range []struct{rrs []dns.RR; tcSection string }{
			{rrs: tc0.Answer}, {rrs: tc0.Ns}, {rrs: resp.Answer, tcSection:"answer"}, {rrs: resp.Ns, tcSection:"ns"},
		}{
			sets := types.SplitSets(section.rrs)
			rrsigs := make(map[types.RRSetKey]*dns.RRSIG)
			for _, rr := range section.rrs {
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
				switch section.tcSection {
				case "answer":
					tc.Answer = append(tc.Answer, rrsig)
				case "ns":
					tc.Ns = append(tc.Ns, rrsig)
				}
			}
		}
		//fmt.Println("dddd")
		sort.Sort(test.RRSet(tc.Answer))
		sort.Sort(test.RRSet(tc.Ns))
		sort.Sort(test.RRSet(tc.Extra))
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
		Initialize:     DefaultDnssecInitialize(),
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
					test.NSEC("nxdomain.x.dnssec_test.com.	100	IN	NSEC	\\000.nxdomain.x.dnssec_test.com. RRSIG NSEC"),
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
		Initialize:     DefaultDnssecInitialize(),
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
					test.NSEC("ml.example. 3600 IN NSEC \\000.ml.example. RRSIG NSEC"),
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
					test.NSEC("ns1.example. 3600 IN NSEC \\000.ns1.example. A CNAME PTR TXT AAAA SRV RRSIG NSEC TLSA CAA"),
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
					test.NSEC("a.z.w.example. 3600 IN NSEC \\000.a.z.w.example. A CNAME PTR MX TXT SRV RRSIG NSEC TLSA CAA"),
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
					test.NSEC("example.	3600	IN	NSEC	\\000.example. A NS SOA PTR MX TXT AAAA SRV RRSIG NSEC TLSA CAA"),
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

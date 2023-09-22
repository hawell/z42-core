package resolver

import (
	"errors"
	"fmt"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"net"
	"strings"
	"testing"
	"time"
	"z42-core/internal/storage"
	"z42-core/internal/types"

	"github.com/miekg/dns"
	"z42-core/internal/test"
)

var handlerTestCases = []*TestCase{
	{
		Name:            "Basic Usage",
		Description:     "Test Basic functionality",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.com."},
		ZoneConfigs:     []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.com.","ns":"ns1.example.com.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"@",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.example.com."},{"host":"ns2.example.com."}]}}`,
				},
				{"x",
					`{
            			"a":{"ttl":300, "records":[{"ip":"1.2.3.4", "country":["ES"]},{"ip":"5.6.7.8", "country":[""]}]},
            			"aaaa":{"ttl":300, "records":[{"ip":"::1"}]},
            			"txt":{"ttl":300, "records":[{"text":"foo"},{"text":"bar"}]},
            			"mx":{"ttl":300, "records":[{"host":"mx1.example.com.", "preference":10},{"host":"mx2.example.com.", "preference":10}]},
            			"srv":{"ttl":300, "records":[{"target":"sip.example.com.","port":555,"priority":10,"weight":100}]}
            		}`,
				},
				{"y",
					`{"cname":{"ttl":300, "host":"x.example.com."}}`,
				},
				{"ns1",
					`{"a":{"ttl":300, "records":[{"ip":"2.2.2.2"}]}}`,
				},
				{"ns2",
					`{"a":{"ttl":300, "records":[{"ip":"3.3.3.3"}]}}`,
				},
				{"_sip._tcp",
					`{"srv":{"ttl":300, "records":[{"target":"sip.example.com.","port":555,"priority":10,"weight":100}]}}`,
				},
				{"_443._tcp.www",
					`{"tlsa":{"ttl":300, "records":[{"usage":0, "selector":0, "matching_type":1, "certificate":"d2abde240d7cd3ee6b4b28c54df034b97983a1d16e8a410e4561cb106618e971"}]}}`,
				},
				{"_990._tcp",
					`{
            			"tlsa":{"ttl":300, "records":[
                			{"usage":1, "selector":1, "matching_type":1, "certificate":"1CFC98A706BCF3683015"},
                			{"usage":1, "selector":1, "matching_type":1, "certificate":"62D5414CD1CC657E3D30"}
						]}
					}`,
				},
				{"sip",
					`{
						"a":{"ttl":300, "records":[{"ip":"7.7.7.7"}]},
            			"aaaa":{"ttl":300, "records":[{"ip":"::1"}]}
					}`,
				},
				{"t.u.v.w",
					`{"a":{"ttl":300, "records":[{"ip":"9.9.9.9"}]}}`,
				},
				{"cnametonx",
					`{"cname":{"ttl":300, "host":"notexists.example.com."}}`,
				},
				{"subdel",
					`{
						"ns":{"ttl":300, "records":[{"host":"ns1.example.com."},{"host":"ns2.example.com."}]},
						"ds":{"ttl":300, "records":[{"key_tag":57855, "algorithm":5, "digest_type":1, "digest":"B6DCD485719ADCA18E5F3D48A2331627FDD3636B"}]}
					}`,
				},
			},
		},
		TestCases: []test.Case{
			// NOAUTH Test
			{
				Desc:  "NOAUTH Test",
				Qname: "dsdsd.sdf.dfd.", Qtype: dns.TypeA,
				Rcode: dns.RcodeNotAuth,
			},
			// A Test
			{
				Desc:  "A Test",
				Qname: "x.example.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("x.example.com. 300 IN A 1.2.3.4"),
					test.A("x.example.com. 300 IN A 5.6.7.8"),
				},
			},
			// AAAA Test
			{
				Desc:  "AAAA Test",
				Qname: "x.example.com.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("x.example.com. 300 IN AAAA ::1"),
				},
			},
			// TXT Test
			{
				Desc:  "TXT Test",
				Qname: "x.example.com.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("x.example.com. 300 IN TXT bar"),
					test.TXT("x.example.com. 300 IN TXT foo"),
				},
			},
			// CNAME Test
			{
				Desc:  "CNAME Test",
				Qname: "y.example.com.", Qtype: dns.TypeCNAME,
				Answer: []dns.RR{
					test.CNAME("y.example.com. 300 IN CNAME x.example.com."),
				},
			},
			// NS Test
			{
				Desc:  "NS Test",
				Qname: "example.com.", Qtype: dns.TypeNS,
				Answer: []dns.RR{
					test.NS("example.com. 300 IN NS ns1.example.com."),
					test.NS("example.com. 300 IN NS ns2.example.com."),
				},
			},
			// MX Test
			{
				Desc:  "MX Test",
				Qname: "x.example.com.", Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("x.example.com. 300 IN MX 10 mx1.example.com."),
					test.MX("x.example.com. 300 IN MX 10 mx2.example.com."),
				},
			},
			// SRV Test
			{
				Desc:  "SRV Test",
				Qname: "_sip._tcp.example.com.", Qtype: dns.TypeSRV,
				Answer: []dns.RR{
					test.SRV("_sip._tcp.example.com. 300 IN SRV 10 100 555 sip.example.com."),
				},
			},
			// TLSA Test
			{
				Desc:  "TLSA Test",
				Qname: "_443._tcp.www.example.com.", Qtype: dns.TypeTLSA,
				Answer: []dns.RR{
					test.TLSA("_443._tcp.www.example.com. 300 IN TLSA 0 0 1 d2abde240d7cd3ee6b4b28c54df034b97983a1d16e8a410e4561cb106618e971"),
				},
			},
			{
				Desc:  "Multiple TLSA Test",
				Qname: "_990._tcp.example.com.", Qtype: dns.TypeTLSA,
				Answer: []dns.RR{
					test.TLSA("_990._tcp.example.com. 300 IN TLSA 1 1 1 1CFC98A706BCF3683015"),
					test.TLSA("_990._tcp.example.com. 300 IN TLSA 1 1 1 62D5414CD1CC657E3D30"),
				},
			},
			// NXDOMAIN Test
			{
				Desc:  "NXDOMAIN Test",
				Qname: "notexists.example.com.", Qtype: dns.TypeA,
				Rcode: dns.RcodeNameError,
				Ns: []dns.RR{
					test.SOA("example.com. 300 IN SOA ns1.example.com. hostmaster.example.com. 1460498836 44 55 66 100"),
				},
			},
			// NXDOMAIN through CNAME Test
			{
				Desc:  "NXDOMAIN through CNAME Test",
				Qname: "cnametonx.example.com.", Qtype: dns.TypeA,
				Rcode: dns.RcodeNameError,
				Answer: []dns.RR{
					test.CNAME("cnametonx.example.com. 300 IN CNAME notexists.example.com."),
				},
				Ns: []dns.RR{
					test.SOA("example.com. 300 IN SOA ns1.example.com. hostmaster.example.com. 1460498836 44 55 66 100"),
				},
			},
			// SOA Test
			{
				Desc:  "SOA Test",
				Qname: "example.com.", Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("example.com. 300 IN SOA ns1.example.com. hostmaster.example.com. 1460498836 44 55 66 100"),
				},
			},
			// DS Test
			{
				Desc:  "DS query",
				Qname: "subdel.example.com.", Qtype: dns.TypeDS,
				Answer: []dns.RR{
					test.DS("subdel.example.com. 300 DS 57855 5 1 B6DCD485719ADCA18E5F3D48A2331627FDD3636B"),
				},
			},
			// not implemented
			{
				Desc:  "unsupported type Test",
				Qname: "example.com.", Qtype: dns.TypeUNSPEC,
				Rcode: dns.RcodeSuccess,
				Ns: []dns.RR{
					test.SOA("example.com. 300 IN SOA ns1.example.com. hostmaster.example.com. 1460498836 44 55 66 100"),
				},
			},
			// Empty non-terminal Test
			{
				Desc:  "non-terminal match",
				Qname: "v.w.example.com.", Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.SOA("example.com. 300 IN SOA ns1.example.com. hostmaster.example.com. 1460498836 44 55 66 100"),
				},
			},
		},
	},
	{
		Name:            "WildCard",
		Description:     "tests related to handling of different wildcard scenarios",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.net."},
		ZoneConfigs:     []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.net.","ns":"ns1.example.net.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"@",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.example.net."},{"host":"ns2.example.net."}]}}`,
				},
				{"sub.*",
					`{"txt":{"ttl":300, "records":[{"text":"this is not a wildcard"}]}}`,
				},
				{"host1",
					`{"a":{"ttl":300, "records":[{"ip":"5.5.5.5"}]}}`,
				},
				{"subdel",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.subdel.example.net."},{"host":"ns2.subdel.example.net."}]}}`,
				},
				{"*",
					`{
						"txt":{"ttl":300, "records":[{"text":"this is a wildcard"}]},
            			"mx":{"ttl":300, "records":[{"host":"host1.example.net.","preference": 10}]}
					}`,
				},
				{"_ssh._tcp.host1",
					`{"srv":{"ttl":300, "records":[{"target":"tcp.example.com.","port":123,"priority":10,"weight":100}]}}`,
				},
				{"_ssh._tcp.host2",
					`{"srv":{"ttl":300, "records":[{"target":"tcp.example.com.","port":123,"priority":10,"weight":100}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "direct name mismatch, wildcard name match, wildcard rr match",
				Qname: "host3.example.net.", Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("host3.example.net. 300 IN MX 10 host1.example.net."),
				},
			},
			{
				Desc:  "direct name mismatch, wildcard name match, wildcard rr mismatch",
				Qname: "host3.example.net.", Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.SOA("example.net. 300 IN SOA ns1.example.net. hostmaster.example.net. 1460498836 44 55 66 100"),
				},
			},
			{
				Desc:  "two level wildcard match",
				Qname: "foo.bar.example.net.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("foo.bar.example.net. 300 IN TXT \"this is a wildcard\""),
				},
			},
			{
				Desc:  "direct name match, direct rr mismatch",
				Qname: "host1.example.net.", Qtype: dns.TypeMX,
				Ns: []dns.RR{
					test.SOA("example.net. 300 IN SOA ns1.example.net. hostmaster.example.net. 1460498836 44 55 66 100"),
				},
			},
			{
				Desc:  "name with * label",
				Qname: "sub.*.example.net.", Qtype: dns.TypeMX,
				Ns: []dns.RR{
					test.SOA("example.net. 300 IN SOA ns1.example.net. hostmaster.example.net. 1460498836 44 55 66 100"),
				},
			},
			{
				Desc:  "name mismatch but non-root parent match",
				Qname: "x.host1.example.net.", Qtype: dns.TypeA,
				Rcode: dns.RcodeNameError,
				Ns: []dns.RR{
					test.SOA("example.net. 300 IN SOA ns1.example.net. hostmaster.example.net. 1460498836 44 55 66 100"),
				},
			},
			{
				Desc:  "qname with * in the middle",
				Qname: "ghost.*.example.net.", Qtype: dns.TypeMX,
				Rcode: dns.RcodeNameError,
				Ns: []dns.RR{
					test.SOA("example.net. 300 IN SOA ns1.example.net. hostmaster.example.net. 1460498836 44 55 66 100"),
				},
			},
			{
				Desc:  "multi level wildcard match",
				Qname: "f.h.g.f.t.r.e.example.net.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("f.h.g.f.t.r.e.example.net. 300 IN TXT \"this is a wildcard\""),
				},
			},
		},
	},
	{
		Name:            "CNAME",
		Description:     "normal cname functionality",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.aaa."},
		ZoneConfigs:     []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.aaa.","ns":"ns1.example.aaa.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"@",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.example.aaa."},{"ttl":300, "host":"ns2.example.aaa."}]}}`,
				},
				{"x",
					`{
						"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]},
                		"aaaa":{"ttl":300, "records":[{"ip":"::1"}]},
                		"txt":{"ttl":300, "records":[{"text":"foo"},{"text":"bar"}]},
                		"mx":{"ttl":300, "records":[{"host":"mx1.example.aaa.", "preference":10},{"host":"mx2.example.aaa.", "preference":10}]},
                		"srv":{"ttl":300, "records":[{"target":"sip.example.aaa.","port":555,"priority":10,"weight":100}]}
					}`,
				},
				{"y",
					`{"cname":{"ttl":300, "host":"x.example.aaa."}}`,
				},
				{"z",
					`{"cname":{"ttl":300, "host":"y.example.aaa."}}`,
				},
				{"w",
					`{
						"a":{"ttl":300, "records":[{"ip":"1.1.1.1"}]},
                		"aaaa":{"ttl":300, "records":[{"ip":"::2"}]},
                		"txt":{"ttl":300, "records":[{"text":"www"},{"text":"qqq"}]},
                		"mx":{"ttl":300, "records":[{"host":"mx3.example.aaa.", "preference":10},{"host":"mx4.example.aaa.", "preference":10}]},
                		"srv":{"ttl":300, "records":[{"target":"sip2.example.aaa.","port":555,"priority":10,"weight":100}]},
						"ns":{"ttl":300, "records":[{"host":"ns1.example.aaa."},{"ttl":300, "host":"ns2.example.aaa."}]},
						"cname":{"ttl":300, "host":"x.example.aaa."}
					}`,
				},
				{"a",
					`{"a":{"ttl":300, "records":[{"ip":"129.0.2.2"}]}}`,
				},
				{"v",
					`{"cname":{"ttl":300, "host":"w.a.example.aaa."}}`,
				},
				{"*.a",
					`{"cname":{"ttl":300, "host":"w.b.example.aaa."}}`,
				},
				{"*.b",
					`{"cname":{"ttl":300, "host":"w.c.example.aaa."}}`,
				},
				{"*.c",
					`{"a":{"ttl":300, "records":[{"ip":"129.0.2.1"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "cname query to existing name without cname",
				Qname: "y.example.aaa.", Qtype: dns.TypeCNAME,
				Answer: []dns.RR{
					test.CNAME("y.example.aaa. 300 IN CNAME x.example.aaa."),
				},
			},
			{
				Desc:  "cname query to existing name with cname",
				Qname: "z.example.aaa.", Qtype: dns.TypeCNAME,
				Answer: []dns.RR{
					test.CNAME("z.example.aaa. 300 IN CNAME y.example.aaa."),
				},
			},
			{
				Desc:  "A query to two level cname to matching name/type",
				Qname: "z.example.aaa.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("x.example.aaa. 300 IN A 1.2.3.4"),
					test.CNAME("y.example.aaa. 300 IN CNAME x.example.aaa."),
					test.CNAME("z.example.aaa. 300 IN CNAME y.example.aaa."),
				},
			},
			{
				Desc:  "AAAA query to two level cname to matching name/type",
				Qname: "z.example.aaa.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("x.example.aaa. 300 IN AAAA ::1"),
					test.CNAME("y.example.aaa. 300 IN CNAME x.example.aaa."),
					test.CNAME("z.example.aaa. 300 IN CNAME y.example.aaa."),
				},
			},
			{
				Desc:  "TXT query to two level cname to matching name/type",
				Qname: "z.example.aaa.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("x.example.aaa. 300 IN TXT bar"),
					test.TXT("x.example.aaa. 300 IN TXT foo"),
					test.CNAME("y.example.aaa. 300 IN CNAME x.example.aaa."),
					test.CNAME("z.example.aaa. 300 IN CNAME y.example.aaa."),
				},
			},
			{
				Desc:  "MX query to two level cname to matching name/type",
				Qname: "z.example.aaa.", Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("x.example.aaa. 300 IN MX 10 mx1.example.aaa."),
					test.MX("x.example.aaa. 300 IN MX 10 mx2.example.aaa."),
					test.CNAME("y.example.aaa. 300 IN CNAME x.example.aaa."),
					test.CNAME("z.example.aaa. 300 IN CNAME y.example.aaa."),
				},
			},
			{
				Desc:  "SRV query to two level cname to matching name/type",
				Qname: "z.example.aaa.", Qtype: dns.TypeSRV,
				Answer: []dns.RR{
					test.SRV("x.example.aaa. 300 IN SRV 10 100 555 sip.example.aaa."),
					test.CNAME("y.example.aaa. 300 IN CNAME x.example.aaa."),
					test.CNAME("z.example.aaa. 300 IN CNAME y.example.aaa."),
				},
			},
			{
				Desc:  "query to cnamed name, name match, type mismatch",
				Qname: "z.example.aaa.", Qtype: dns.TypeNS,
				Answer: []dns.RR{
					test.CNAME("y.example.aaa. 300 IN CNAME x.example.aaa."),
					test.CNAME("z.example.aaa. 300 IN CNAME y.example.aaa."),
				},
				Ns: []dns.RR{
					test.SOA("example.aaa. 300 IN SOA ns1.example.aaa. hostmaster.example.aaa. 1460498836 44 55 66 100"),
				},
			},
			{
				Desc:  "A query to a name containing cname rr",
				Qname: "w.example.aaa.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("w.example.aaa. 300 IN CNAME x.example.aaa."),
					test.A("x.example.aaa. 300 IN A 1.2.3.4"),
				},
			},
			{
				Desc:  "AAAA query to a name containing cname rr",
				Qname: "w.example.aaa.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.CNAME("w.example.aaa. 300 IN CNAME x.example.aaa."),
					test.AAAA("x.example.aaa. 300 IN AAAA ::1"),
				},
			},
			{
				Desc:  "TXT query to a name containing cname rr",
				Qname: "w.example.aaa.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.CNAME("w.example.aaa. 300 IN CNAME x.example.aaa."),
					test.TXT("x.example.aaa. 300 IN TXT bar"),
					test.TXT("x.example.aaa. 300 IN TXT foo"),
				},
			},
			{
				Desc:  "NS query to a name containing cname rr",
				Qname: "w.example.aaa.", Qtype: dns.TypeNS,
				Answer: []dns.RR{
					test.CNAME("w.example.aaa. 300 IN CNAME x.example.aaa."),
				},
				Ns: []dns.RR{
					test.SOA("example.aaa. 300 IN SOA ns1.example.aaa. hostmaster.example.aaa. 1460498836 44 55 66 100"),
				},
			},
			{
				Desc:  "MX query to a name containing cname rr",
				Qname: "w.example.aaa.", Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.CNAME("w.example.aaa. 300 IN CNAME x.example.aaa."),
					test.MX("x.example.aaa. 300 IN MX 10 mx1.example.aaa."),
					test.MX("x.example.aaa. 300 IN MX 10 mx2.example.aaa."),
				},
			},
			{
				Desc:  "SRV query to a name containing cname rr",
				Qname: "w.example.aaa.", Qtype: dns.TypeSRV,
				Answer: []dns.RR{
					test.CNAME("w.example.aaa. 300 IN CNAME x.example.aaa."),
					test.SRV("x.example.aaa. 300 IN SRV 10 100 555 sip.example.aaa."),
				},
			},
			{
				Desc:  "cname + wildcard",
				Qname: "v.example.aaa.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("v.example.aaa. 300 IN CNAME w.a.example.aaa."),
					test.CNAME("w.a.example.aaa. 300 IN CNAME w.b.example.aaa."),
					test.CNAME("w.b.example.aaa. 300 IN CNAME w.c.example.aaa."),
					test.A("w.c.example.aaa. 300 IN	A 129.0.2.1"),
				},
			},
		},
	},
	{
		Name:            "empty values",
		Description:     "test handler behaviour with empty records",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.bbb."},
		ZoneConfigs:     []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.bbb.","ns":"ns1.example.bbb.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"x",
					`{}`,
				},
				{"y",
					`{"cname":{"ttl":300, "host":"x.example.bbb."}}`,
				},
				{"z",
					`{}`,
				},
			},
		},
		TestCases: []test.Case{
			// empty A test
			{
				Desc:  "empty A test",
				Qname: "z.example.bbb.", Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty AAAA test
			{
				Desc:  "empty AAAA test",
				Qname: "z.example.bbb.", Qtype: dns.TypeAAAA,
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty TXT test
			{
				Desc:  "empty TXT test",
				Qname: "z.example.bbb.", Qtype: dns.TypeTXT,
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty NS test
			{
				Desc:  "empty NS test",
				Qname: "z.example.bbb.", Qtype: dns.TypeNS,
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty MX test
			{
				Desc:  "empty MX test",
				Qname: "z.example.bbb.", Qtype: dns.TypeMX,
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty SRV test
			{
				Desc:  "empty SRV test",
				Qname: "z.example.bbb.", Qtype: dns.TypeSRV,
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty CNAME test
			{
				Desc:  "empty CNAME test",
				Qname: "x.example.bbb.", Qtype: dns.TypeCNAME,
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty A test with cname
			{
				Desc:  "empty A test with cname",
				Qname: "y.example.bbb.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("y.example.bbb.	300	IN	CNAME	x.example.bbb."),
				},
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty AAAA test with cname
			{
				Desc:  "empty AAAA test with cname",
				Qname: "y.example.bbb.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.CNAME("y.example.bbb.	300	IN	CNAME	x.example.bbb."),
				},
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty TXT test with cname
			{
				Desc:  "empty TXT test with cname",
				Qname: "y.example.bbb.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.CNAME("y.example.bbb.	300	IN	CNAME	x.example.bbb."),
				},
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty NS test with cname
			{
				Desc:  "empty NS test with cname",
				Qname: "y.example.bbb.", Qtype: dns.TypeNS,
				Answer: []dns.RR{
					test.CNAME("y.example.bbb.	300	IN	CNAME	x.example.bbb."),
				},
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty MX test with cname
			{
				Desc:  "empty MX test with cname",
				Qname: "y.example.bbb.", Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.CNAME("y.example.bbb.	300	IN	CNAME	x.example.bbb."),
				},
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
			// empty SRV test with cname
			{
				Desc:  "empty SRV test with cname",
				Qname: "y.example.bbb.", Qtype: dns.TypeSRV,
				Answer: []dns.RR{
					test.CNAME("y.example.bbb.	300	IN	CNAME	x.example.bbb."),
				},
				Ns: []dns.RR{
					test.SOA("example.bbb. 300 IN SOA ns1.example.bbb. hostmaster.example.bbb. 1460498836 44 55 66 100"),
				},
			},
		},
	},
	{
		Name:            "long text",
		Description:     "text field longer than 255 bytes",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.ccc."},
		ZoneConfigs:     []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.ccc.","ns":"ns1.example.ccc.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"x",
					`{"txt":{"ttl":300, "records":[{"text":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "long TXT value",
				Qname: "x.example.ccc.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("x.example.ccc. 300 IN TXT \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\""),
				},
			},
		},
	},
	{
		Name:            "cname flattening",
		Description:     "eliminate intermediate cname records when cname flattening is enabled",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.ddd."},
		ZoneConfigs:     []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.ddd.","ns":"ns1.example.ddd.","refresh":44,"retry":55,"expire":66},"cname_flattening":true}`},
		Entries: [][][]string{
			{
				{"@",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.example.ddd."},{"ttl":300, "host":"ns2.example.ddd."}]}}`,
				},
				{"a",
					`{
						"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]},
                		"aaaa":{"ttl":300, "records":[{"ip":"::1"}]},
                		"txt":{"ttl":300, "records":[{"text":"foo"},{"text":"bar"}]},
                		"mx":{"ttl":300, "records":[{"host":"mx1.example.ddd.", "preference":10},{"host":"mx2.example.ddd.", "preference":10}]},
                		"srv":{"ttl":300, "records":[{"target":"sip.example.ddd.","port":555,"priority":10,"weight":100}]}
					}`,
				},
				{"b",
					`{"cname":{"ttl":300, "host":"a.example.ddd."}}`,
				},
				{"c",
					`{"cname":{"ttl":300, "host":"b.example.ddd."}}`,
				},
				{"d",
					`{"cname":{"ttl":300, "host":"c.example.ddd."}}`,
				},
				{"e",
					`{"cname":{"ttl":300, "host":"d.example.ddd."}}`,
				},
				{"x",
					`{"a":{"ttl":300, "records":[{"ip":"129.0.2.2"}]}}`,
				},
				{"w",
					`{"cname":{"ttl":300, "host":"w.x.example.ddd."}}`,
				},
				{"*.x",
					`{"cname":{"ttl":300, "host":"w.y.example.ddd."}}`,
				},
				{"*.y",
					`{"cname":{"ttl":300, "host":"w.z.example.ddd."}}`,
				},
				{"*.z",
					`{"a":{"ttl":300, "records":[{"ip":"129.0.2.1"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "multi cname, A query to existing name/type",
				Qname: "e.example.ddd.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("e.example.ddd. 300 IN A 1.2.3.4"),
				},
			},
			{
				Desc:  "multi cname, AAAA query to existing name/type",
				Qname: "e.example.ddd.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("e.example.ddd. 300 IN AAAA ::1"),
				},
			},
			{
				Desc:  "multi cname, TXT query to existing name/type",
				Qname: "e.example.ddd.", Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT("e.example.ddd. 300 IN TXT \"bar\""),
					test.TXT("e.example.ddd. 300 IN TXT \"foo\""),
				},
			},
			// MX Test
			{
				Desc:  "multi cname, MX query to existing name/type",
				Qname: "e.example.ddd.", Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("e.example.ddd. 300 IN MX 10 mx1.example.ddd."),
					test.MX("e.example.ddd. 300 IN MX 10 mx2.example.ddd."),
				},
			},
			// SRV Test
			{
				Desc:  "multi cname, SRV query to existing name/type",
				Qname: "e.example.ddd.", Qtype: dns.TypeSRV,
				Answer: []dns.RR{
					test.SRV("e.example.ddd. 300 IN SRV 10 100 555 sip.example.ddd."),
				},
			},
			{
				Desc:  "cname query to a name containing cname",
				Qname: "e.example.ddd.", Qtype: dns.TypeCNAME,
				Answer: []dns.RR{
					test.CNAME("e.example.ddd. 300 IN CNAME d.example.ddd."),
				},
			},
			{
				Desc:  "cname with wildcard",
				Qname: "w.example.ddd.",
				Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("w.example.ddd.	300	IN	A	129.0.2.1"),
				},
			},
		},
	},
	{
		Name:            "caa test",
		Description:     "basic caa functionality",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.caa.", "nocaa.caa."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.caa.","ns":"ns1.example.caa.","refresh":44,"retry":55,"expire":66}}`,
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.nocaa.caa.","ns":"ns1.nocaa.caa.","refresh":44,"retry":55,"expire":66}}`,
		},
		Entries: [][][]string{
			{
				{"@",
					`{"caa":{"ttl":300, "records":[{"tag":"issue", "value":"godaddy.com;", "flag":0}]}}`,
				},
				{"a.b.c.d",
					`{"cname":{"ttl":300, "host":"b.c.d.example.caa."}}`,
				},
				{"b.c.d",
					`{"cname":{"ttl":300, "host":"c.d.example.caa."}}`,
				},
				{"c.d",
					`{"cname":{"ttl":300, "host":"d.example.caa."}}`,
				},
				{"d",
					`{"cname":{"ttl":300, "host":"example.caa."}}`,
				},
				{"x.y.z",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"y.z",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"z",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"h",
					`{"caa":{"ttl":300, "records":[{"tag":"issue", "value":"godaddy2.com;", "flag":0}]}}`,
				},
				{"g.h",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"j.g.h",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
			},
			{
				{"@",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"www2",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"www3",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "direct caa match",
				Qname: "example.caa.", Qtype: dns.TypeCAA,
				Answer: []dns.RR{
					test.CAA("example.caa.	300	IN	CAA	0 issue \"godaddy.com;\""),
				},
			},
			{
				Desc:  "caa through cname",
				Qname: "a.b.c.d.example.caa.", Qtype: dns.TypeCAA,
				Answer: []dns.RR{
					test.CNAME("a.b.c.d.example.caa. 300 IN CNAME b.c.d.example.caa."),
					test.CNAME("b.c.d.example.caa. 300 IN CNAME c.d.example.caa."),
					test.CNAME("c.d.example.caa. 300 IN CNAME d.example.caa."),
					test.CNAME("d.example.caa. 300 IN CNAME example.caa."),
					test.CAA("example.caa. 300 IN CAA 0 issue \"godaddy.com;\""),
				},
			},
			{
				Desc:  "match with root caa",
				Qname: "x.y.z.example.caa.", Qtype: dns.TypeCAA,
				Answer: []dns.RR{
					test.CAA("x.y.z.example.caa.	300	IN	CAA	0 issue \"godaddy.com;\""),
				},
			},
			{
				Desc:  "direct subdomain caa match",
				Qname: "h.example.caa.", Qtype: dns.TypeCAA,
				Answer: []dns.RR{
					test.CAA("h.example.caa.	300	IN	CAA	0 issue \"godaddy2.com;\""),
				},
			},
			{
				Desc:  "match with non-root parent caa",
				Qname: "g.h.example.caa.", Qtype: dns.TypeCAA,
				Answer: []dns.RR{
					test.CAA("g.h.example.caa.	300	IN	CAA	0 issue \"godaddy2.com;\""),
				},
			},
			{
				Desc:  "multi level match with non-root parent caa",
				Qname: "j.g.h.example.caa.", Qtype: dns.TypeCAA,
				Answer: []dns.RR{
					test.CAA("j.g.h.example.caa.	300	IN	CAA	0 issue \"godaddy2.com;\""),
				},
			},
			{
				Desc:  "no caa root",
				Qname: "nocaa.caa.", Qtype: dns.TypeCAA,
				Ns: []dns.RR{
					test.SOA("nocaa.caa.	300	IN	SOA	ns1.nocaa.caa. hostmaster.nocaa.caa. 1570970363 44 55 66 100"),
				},
			},
			{
				Desc:  "no caa subdomain",
				Qname: "www.nocaa.caa.", Qtype: dns.TypeCAA,
				Ns: []dns.RR{
					test.SOA("nocaa.caa.	300	IN	SOA	ns1.nocaa.caa. hostmaster.nocaa.caa. 1570970363 44 55 66 100"),
				},
			},
			{
				Desc:  "no caa subdomain",
				Qname: "www2.nocaa.caa.", Qtype: dns.TypeCAA,
				Ns: []dns.RR{
					test.SOA("nocaa.caa.	300	IN	SOA	ns1.nocaa.caa. hostmaster.nocaa.caa. 1570970363 44 55 66 100"),
				},
			},
			{
				Desc:  "no caa subdomain",
				Qname: "www3.nocaa.caa.", Qtype: dns.TypeCAA,
				Ns: []dns.RR{
					test.SOA("nocaa.caa.	300	IN	SOA	ns1.nocaa.caa. hostmaster.nocaa.caa. 1570970363 44 55 66 100"),
				},
			},
		},
	},
	{
		Name:            "PTR test",
		Description:     "basic ptr functionality",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"0.0.127.in-addr.arpa.", "20.127.10.in-addr.arpa."},
		ZoneConfigs:     []string{"", ""},
		Entries: [][][]string{
			{
				{"1",
					`{"ptr":{"ttl":300, "domain":"localhost"}}`,
				},
			},
			{
				{"54",
					`{"ptr":{"ttl":300, "domain":"example.fff"}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "1.0.0.127.in-addr.arpa.", Qtype: dns.TypePTR,
				Answer: []dns.RR{
					test.PTR("1.0.0.127.in-addr.arpa. 300 IN PTR localhost."),
				},
			},
			{
				Qname: "54.20.127.10.in-addr.arpa.", Qtype: dns.TypePTR,
				Answer: []dns.RR{
					test.PTR("54.20.127.10.in-addr.arpa. 300 IN PTR example.fff."),
				},
			},
		},
	},
	{
		Name:            "ANAME test",
		Description:     "test aname functionality",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.com."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.com.","ns":"ns1.example.com.","refresh":44,"retry":55,"expire":66}}`,
		},
		Entries: [][][]string{
			{
				{"@",
					`{"aname":{"location":"aname.example.com."}}`,
				},
				{"nxlocal",
					`{"aname":{"location":"nx.example.com."}}`,
				},
				{"empty",
					`{"aname":{"location":"e.example.com."}}`,
				},
				{"e",
					`{"txt":{"ttl":300, "records":[{"text":"foo"}]}}`,
				},
				{"upstream",
					`{"aname":{"location":"dns.msftncsi.com."}}`,
				},
				{"nxupstream",
					`{"aname":{"location":"anamex.xmpl.an."}}`,
				},
				{"aname",
					`{"a":{"ttl":300, "records":[{"ip":"6.5.6.5"}]}, "aaaa":{"ttl":300, "records":[{"ip":"::1"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "A aname at root",
				Qname: "example.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("example.com. 300 IN A 6.5.6.5"),
				},
			},
			{
				Desc:  "AAAA aname at root",
				Qname: "example.com.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("example.com. 300 IN AAAA ::1"),
				},
			},
			{
				Desc:  "A aname at subdomain",
				Qname: "upstream.example.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("upstream.example.com. 303 IN A 131.107.255.255"),
				},
			},
			{
				Desc:  "AAAA aname at subdomain",
				Qname: "upstream.example.com.", Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("upstream.example.com. 303 IN AAAA fd3e:4f5a:5b81::1"),
				},
			},
			{
				Desc:  "A aname to nx local",
				Qname: "nxlocal.example.com.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
			{
				Desc:  "AAAA aname to nx local",
				Qname: "nxlocal.example.com.", Qtype: dns.TypeAAAA,
				Rcode: dns.RcodeServerFailure,
			},
			{
				Desc:  "A aname to empty location",
				Qname: "empty.example.com.", Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.SOA("example.com.	300	IN	SOA	ns1.example.com. hostmaster.example.com. 1570970363 44 55 66 100"),
				},
			},
			{
				Desc:  "AAAA aname to empty location",
				Qname: "empty.example.com.", Qtype: dns.TypeAAAA,
				Ns: []dns.RR{
					test.SOA("example.com.	300	IN	SOA	ns1.example.com. hostmaster.example.com. 1570970363 44 55 66 100"),
				},
			},
			{
				Desc:  "A aname to nx external",
				Qname: "nxupstream.example.com.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
			{
				Desc:  "AAAA aname to nx external",
				Qname: "nxupstream.example.com.", Qtype: dns.TypeAAAA,
				Rcode: dns.RcodeServerFailure,
			},
		},
	},
	{
		Name:            "weighted aname test",
		Description:     "weight filter should be applied on aname results as well",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify: func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
			RegisterTestingT(t)
			ipsCount := []int{0, 0, 0}
			for i := 0; i < 1000; i++ {
				r := testCase.TestCases[0].Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg
				Expect(resp.Rcode).To(Equal(dns.RcodeSuccess))
				Expect(len(resp.Answer)).NotTo(Equal(0))
				a := resp.Answer[0].(*dns.A)
				switch a.A.String() {
				case "1.1.1.1":
					ipsCount[0]++
				case "2.2.2.2":
					ipsCount[1]++
				case "3.3.3.3":
					ipsCount[2]++
				default:
					Expect(true).To(BeFalse())
					fmt.Println("invalid ip : ", a.A.String())
				}
			}
			Expect(ipsCount[0] < ipsCount[1] && ipsCount[1] < ipsCount[2]).To(BeTrue())
			ipsCount = []int{0, 0, 0}
			for i := 0; i < 1000; i++ {
				r := testCase.TestCases[1].Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg
				Expect(resp.Rcode).To(Equal(dns.RcodeSuccess))
				Expect(len(resp.Answer)).NotTo(Equal(0))
				aaaa := resp.Answer[0].(*dns.AAAA)
				switch aaaa.AAAA.String() {
				case "2001:db8::1":
					ipsCount[0]++
				case "2001:db8::2":
					ipsCount[1]++
				case "2001:db8::3":
					ipsCount[2]++
				default:
					Expect(true).To(BeFalse())
					fmt.Println("invalid ip : ", aaaa.AAAA.String())
				}
			}
			Expect(ipsCount[0] < ipsCount[1] && ipsCount[1] < ipsCount[2]).To(BeTrue())
		},
		Zones: []string{"example.com."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.com.","ns":"ns1.example.com.","refresh":44,"retry":55,"expire":66}}`,
		},
		Entries: [][][]string{
			{
				{"upstream2",
					`{"aname":{"location":"aname2.example.com."}}`,
				},
				{"aname2",
					`{
						"a":{"ttl":300, "filter": {"count":"single", "order": "weighted", "geo_filter":"none"}, "records":[{"ip":"1.1.1.1", "weight":1},{"ip":"2.2.2.2", "weight":5},{"ip":"3.3.3.3", "weight":10}]},
						"aaaa":{"ttl":300, "filter": {"count":"single", "order": "weighted", "geo_filter":"none"}, "records":[{"ip":"2001:db8::1", "weight":1},{"ip":"2001:db8::2", "weight":5},{"ip":"2001:db8::3", "weight":10}]}
					}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "A to weighted aname",
				Qname: "upstream2.example.com.", Qtype: dns.TypeA,
			},
			{
				Desc:  "AAAA to weighted aname",
				Qname: "upstream2.example.com.", Qtype: dns.TypeAAAA,
			},
		},
	},
	{
		Name:            "geofilter test",
		Description:     "test various geofilter scenarios",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify: func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
			RegisterTestingT(t)
			var filterGeoSourceIps = []string{
				"127.0.0.1",
				"127.0.0.1",
				"127.0.0.1",
				"127.0.0.1",
				"127.0.0.1",
				"94.76.229.204",  // country = GB
				"154.11.253.242", // location = CA near US
				"212.83.32.45",   // ASN = 47447
				"212.83.32.45",   // country = DE, ASN = 47447
				"212.83.32.45",
				"178.18.89.144",
				"127.0.0.1",
				"213.95.10.76",   // DE
				"94.76.229.204",  // GB
				"154.11.253.242", // CA
				"127.0.0.1",
			}
			for i, tc := range testCase.TestCases {
				sa := filterGeoSourceIps[i]
				opt := &dns.OPT{
					Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT, Class: dns.ClassANY, Rdlength: 0, Ttl: 300},
					Option: []dns.EDNS0{
						&dns.EDNS0_SUBNET{
							Address:       net.ParseIP(sa),
							Code:          dns.EDNS0SUBNET,
							Family:        1,
							SourceNetmask: 32,
							SourceScope:   0,
						},
					},
				}
				r := tc.Msg()
				r.Extra = append(r.Extra, opt)
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg
				resp.Extra = nil

				err := test.SortAndCheck(resp, tc)
				Expect(err).To(BeNil())
			}
		},
		Zones:       []string{"filtergeo.com."},
		ZoneConfigs: []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.filter.com.","ns":"ns1.filter.com.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"ww1",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":[""]},
            				{"ip":"127.0.0.2", "country":[""]},
            				{"ip":"127.0.0.3", "country":[""]},
            				{"ip":"127.0.0.4", "country":[""]},
            				{"ip":"127.0.0.5", "country":[""]},
            				{"ip":"127.0.0.6", "country":[""]}
						],
            			"filter":{"count":"multi","order":"none","geo_filter":"none"}}
					}`,
				},
				{"ww2",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":["US"]},
            				{"ip":"127.0.0.2", "country":["GB"]},
            				{"ip":"127.0.0.3", "country":["ES"]},
							{"ip":"127.0.0.4", "country":[""]},
            				{"ip":"127.0.0.5", "country":[""]},
            				{"ip":"127.0.0.6", "country":[""]}
						],
            			"filter":{"count":"multi","order":"none","geo_filter":"country"}}
					}`,
				},
				{"ww3",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"192.30.252.225"},
            				{"ip":"94.76.229.204"},
            				{"ip":"84.88.14.229"},
							{"ip":"192.168.0.1"}
						],
            			"filter":{"count":"multi","order":"none","geo_filter":"location"}}
					}`,
				},
				{"ww4",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "asn":[47447]},
            				{"ip":"127.0.0.2", "asn":[20776]},
            				{"ip":"127.0.0.3", "asn":[35470]},
            				{"ip":"127.0.0.4", "asn":[0]},
            				{"ip":"127.0.0.5", "asn":[0]},
            				{"ip":"127.0.0.6", "asn":[0]}
						],
        				"filter":{"count":"multi", "order":"none","geo_filter":"asn"}}
					}`,
				},
				{"ww5",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":["DE"], "asn":[47447]},
            				{"ip":"127.0.0.2", "country":["DE"], "asn":[20776]},
            				{"ip":"127.0.0.3", "country":["DE"], "asn":[35470]},
            				{"ip":"127.0.0.4", "country":["GB"], "asn":[0]},
            				{"ip":"127.0.0.5", "country":[""], "asn":[0]},
            				{"ip":"127.0.0.6", "country":[""], "asn":[0]}
						],
        				"filter":{"count":"multi", "order":"none","geo_filter":"asn+country"}}
					}`,
				},
				{"ww6",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "asn":[47447,20776]},
            				{"ip":"127.0.0.2", "asn":[0,35470]},
            				{"ip":"127.0.0.3", "asn":[35470]},
            				{"ip":"127.0.0.4", "asn":[0]},
            				{"ip":"127.0.0.5", "asn":[]},
            				{"ip":"127.0.0.6"}
						],
        				"filter":{"count":"multi", "order":"none","geo_filter":"asn"}}
					}`,
				},
				{"ww7",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":["DE", "GB"]},
            				{"ip":"127.0.0.2", "country":["", "DE"]},
            				{"ip":"127.0.0.3", "country":["DE"]},
            				{"ip":"127.0.0.4", "country":["CA"]},
            				{"ip":"127.0.0.5", "country": [""]},
            				{"ip":"127.0.0.6", "country": []},
            				{"ip":"127.0.0.7"}
						],
        				"filter":{"count":"multi", "order":"none","geo_filter":"country"}}
					}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "ww1.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww1.filtergeo.com. 300 IN A 127.0.0.1"),
					test.A("ww1.filtergeo.com. 300 IN A 127.0.0.2"),
					test.A("ww1.filtergeo.com. 300 IN A 127.0.0.3"),
					test.A("ww1.filtergeo.com. 300 IN A 127.0.0.4"),
					test.A("ww1.filtergeo.com. 300 IN A 127.0.0.5"),
					test.A("ww1.filtergeo.com. 300 IN A 127.0.0.6"),
				},
			},
			{
				Qname: "ww2.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww2.filtergeo.com. 300 IN A 127.0.0.4"),
					test.A("ww2.filtergeo.com. 300 IN A 127.0.0.5"),
					test.A("ww2.filtergeo.com. 300 IN A 127.0.0.6"),
				},
			},
			{
				Qname: "ww3.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww3.filtergeo.com. 300 IN A 192.168.0.1"),
				},
			},
			{
				Qname: "ww4.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww4.filtergeo.com. 300 IN A 127.0.0.4"),
					test.A("ww4.filtergeo.com. 300 IN A 127.0.0.5"),
					test.A("ww4.filtergeo.com. 300 IN A 127.0.0.6"),
				},
			},
			{
				Qname: "ww5.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww5.filtergeo.com. 300 IN A 127.0.0.5"),
					test.A("ww5.filtergeo.com. 300 IN A 127.0.0.6"),
				},
			},
			{
				Qname: "ww2.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww2.filtergeo.com. 300 IN A 127.0.0.2"),
				},
			},
			{
				Qname: "ww3.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww3.filtergeo.com. 300 IN A 192.30.252.225"),
				},
			},
			{
				Qname: "ww4.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww4.filtergeo.com. 300 IN A 127.0.0.1"),
				},
			},
			{
				Qname: "ww5.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww5.filtergeo.com. 300 IN A 127.0.0.1"),
				},
			},
			{
				Qname: "ww6.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww6.filtergeo.com. 300 IN A 127.0.0.1"),
				},
			},
			{
				Qname: "ww6.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww6.filtergeo.com. 300 IN A 127.0.0.2"),
					test.A("ww6.filtergeo.com. 300 IN A 127.0.0.3"),
				},
			},
			{
				Qname: "ww6.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww6.filtergeo.com. 300 IN A 127.0.0.2"),
					test.A("ww6.filtergeo.com. 300 IN A 127.0.0.4"),
					test.A("ww6.filtergeo.com. 300 IN A 127.0.0.5"),
					test.A("ww6.filtergeo.com. 300 IN A 127.0.0.6"),
				},
			},
			{
				Qname: "ww7.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.1"),
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.2"),
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.3"),
				},
			},
			{
				Qname: "ww7.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.1"),
				},
			},
			{
				Qname: "ww7.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.4"),
				},
			},
			{
				Qname: "ww7.filtergeo.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.2"),
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.5"),
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.6"),
					test.A("ww7.filtergeo.com. 300 IN A 127.0.0.7"),
				},
			},
		},
	},
	{
		Name:            "filter multi ip",
		Description:     "ip filter functionality for multiple value results",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify: func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
			RegisterTestingT(t)
			for i := 0; i < 10; i++ {
				tc := testCase.TestCases[0]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg

				err := test.SortAndCheck(resp, tc)
				Expect(err).To(BeNil())
			}

			w1, w4, w10, w2, w20 := 0, 0, 0, 0, 0
			for i := 0; i < 10000; i++ {
				tc := testCase.TestCases[1]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg
				Expect(len(resp.Answer)).To(Equal(5))

				resa := resp.Answer[0].(*dns.A)

				switch resa.A.String() {
				case "127.0.0.1":
					w1++
				case "127.0.0.2":
					w4++
				case "127.0.0.3":
					w10++
				case "127.0.0.4":
					w2++
				case "127.0.0.5":
					w20++
				}
			}
			// fmt.Println(w1, w2, w4, w10, w20)
			Expect(w1 <= w2).To(BeTrue())
			Expect(w2 <= w4).To(BeTrue())
			Expect(w4 <= w10).To(BeTrue())
			Expect(w10 <= w20).To(BeTrue())

			rr := make([]int, 5)
			for i := 0; i < 10000; i++ {
				tc := testCase.TestCases[2]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg
				Expect(len(resp.Answer)).To(Equal(5))

				resa := resp.Answer[0].(*dns.A)

				switch resa.A.String() {
				case "127.0.0.1":
					rr[0]++
				case "127.0.0.2":
					rr[1]++
				case "127.0.0.3":
					rr[2]++
				case "127.0.0.4":
					rr[3]++
				case "127.0.0.5":
					rr[4]++
				}
			}
			// fmt.Println(rr)
			for i := range rr {
				Expect(rr[i] >= 1500).To(BeTrue())
				Expect(rr[i] <= 2500).To(BeTrue())
			}
		},
		Zones:       []string{"filtermulti.com."},
		ZoneConfigs: []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.filtermulti.com.","ns":"ns1.filter.com.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"ww1",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":[""]},
            				{"ip":"127.0.0.2", "country":[""]},
            				{"ip":"127.0.0.3", "country":[""]},
            				{"ip":"127.0.0.4", "country":[""]},
            				{"ip":"127.0.0.5", "country":[""]},
            				{"ip":"127.0.0.6", "country":[""]}
						],
            			"filter":{"count":"multi","order":"none","geo_filter":"none"}}
					}`,
				},
				{"ww2",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":[""], "weight":1},
            				{"ip":"127.0.0.2", "country":[""], "weight":4},
            				{"ip":"127.0.0.3", "country":[""], "weight":10},
            				{"ip":"127.0.0.4", "country":[""], "weight":2},
            				{"ip":"127.0.0.5", "country":[""], "weight":20}
						],
            			"filter":{"count":"multi","order":"weighted","geo_filter":"none"}}
					}`,
				},
				{"ww3",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":[""]},
            				{"ip":"127.0.0.2", "country":[""]},
            				{"ip":"127.0.0.3", "country":[""]},
            				{"ip":"127.0.0.4", "country":[""]},
            				{"ip":"127.0.0.5", "country":[""]}
						],
            			"filter":{"count":"multi","order":"rr","geo_filter":"none"}}
					}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "ww1.filtermulti.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww1.filtermulti.com. 300 IN A 127.0.0.1"),
					test.A("ww1.filtermulti.com. 300 IN A 127.0.0.2"),
					test.A("ww1.filtermulti.com. 300 IN A 127.0.0.3"),
					test.A("ww1.filtermulti.com. 300 IN A 127.0.0.4"),
					test.A("ww1.filtermulti.com. 300 IN A 127.0.0.5"),
					test.A("ww1.filtermulti.com. 300 IN A 127.0.0.6"),
				},
			},
			{
				Qname: "ww2.filtermulti.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{},
			},
			{
				Qname: "ww3.filtermulti.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{},
			},
		},
	},
	{
		Name:            "filter single ip",
		Description:     "ip filter functionality for single value results",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify: func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
			RegisterTestingT(t)
			for i := 0; i < 10; i++ {
				tc := testCase.TestCases[0]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg

				err := test.SortAndCheck(resp, tc)
				Expect(err).To(BeNil())
			}

			w1, w4, w10, w2, w20 := 0, 0, 0, 0, 0
			for i := 0; i < 10000; i++ {
				tc := testCase.TestCases[1]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg
				Expect(len(resp.Answer)).To(Equal(1))

				resa := resp.Answer[0].(*dns.A)

				switch resa.A.String() {
				case "127.0.0.1":
					w1++
				case "127.0.0.2":
					w4++
				case "127.0.0.3":
					w10++
				case "127.0.0.4":
					w2++
				case "127.0.0.5":
					w20++
				}
			}
			// fmt.Println(w1, w2, w4, w10, w20)
			Expect(w1 <= w2).To(BeTrue())
			Expect(w2 <= w4).To(BeTrue())
			Expect(w4 <= w10).To(BeTrue())
			Expect(w10 <= w20).To(BeTrue())

			rr := make([]int, 5)
			for i := 0; i < 10000; i++ {
				tc := testCase.TestCases[2]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg
				Expect(len(resp.Answer)).To(Equal(1))

				resa := resp.Answer[0].(*dns.A)

				switch resa.A.String() {
				case "127.0.0.1":
					rr[0]++
				case "127.0.0.2":
					rr[1]++
				case "127.0.0.3":
					rr[2]++
				case "127.0.0.4":
					rr[3]++
				case "127.0.0.5":
					rr[4]++
				}
			}
			// fmt.Println(rr)
			for i := range rr {
				Expect(rr[i] >= 1500).To(BeTrue())
				Expect(rr[i] <= 2500).To(BeTrue())
			}
		},
		Zones:       []string{"filtersingle.com."},
		ZoneConfigs: []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.filtersingle.com.","ns":"ns1.filter.com.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"ww1",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":[""]},
            				{"ip":"127.0.0.2", "country":[""]},
            				{"ip":"127.0.0.3", "country":[""]},
            				{"ip":"127.0.0.4", "country":[""]},
            				{"ip":"127.0.0.5", "country":[""]},
            				{"ip":"127.0.0.6", "country":[""]}
						],
            			"filter":{"count":"single","order":"none","geo_filter":"none"}}
					}`,
				},
				{"ww2",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":[""], "weight":1},
            				{"ip":"127.0.0.2", "country":[""], "weight":4},
            				{"ip":"127.0.0.3", "country":[""], "weight":10},
            				{"ip":"127.0.0.4", "country":[""], "weight":2},
            				{"ip":"127.0.0.5", "country":[""], "weight":20}
						],
            			"filter":{"count":"single","order":"weighted","geo_filter":"none"}}
					}`,
				},
				{"ww3",
					`{
						"a":{"ttl":300, "records":[
            				{"ip":"127.0.0.1", "country":[""]},
            				{"ip":"127.0.0.2", "country":[""]},
            				{"ip":"127.0.0.3", "country":[""]},
            				{"ip":"127.0.0.4", "country":[""]},
            				{"ip":"127.0.0.5", "country":[""]}
						],
            			"filter":{"count":"single","order":"rr","geo_filter":"none"}}
					}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "ww1.filtersingle.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ww1.filtersingle.com. 300 IN A 127.0.0.1"),
				},
			},
			{
				Qname: "ww2.filtersingle.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{},
			},
			{
				Qname: "ww3.filtersingle.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{},
			},
		},
	},
	{
		Name:            "cname upstream",
		Description:     "cname should not leave authoritative zone",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify: func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
			RegisterTestingT(t)
			tc := testCase.TestCases[0]
			r := tc.Msg()
			w := test.NewRecorder(&test.ResponseWriter{})
			state := NewRequestContext(w, r)
			handler.HandleRequest(state)

			resp := w.Msg
			// fmt.Println(resp)
			Expect(resp.Rcode).To(Equal(dns.RcodeSuccess))
			cname := resp.Answer[0].(*dns.CNAME)
			Expect(cname.Target).To(Equal("www.google.com."))
		},
		Zones:       []string{"upstreamcname.com."},
		ZoneConfigs: []string{`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.upstreamcname.com.","ns":"ns1.upstreamcname.com.","refresh":44,"retry":55,"expire":66}}`},
		Entries: [][][]string{
			{
				{"upstream",
					`{"cname":{"ttl":300, "host":"www.google.com"}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "upstream.upstreamcname.com.", Qtype: dns.TypeA,
			},
		},
	},
	{
		Name:            "cname outside domain",
		Description:     "should not follow cname between authoritative zones",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"inside.cnm.", "outside.cnm.", "flattening.cnm."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.inside.cnm.","ns":"ns1.inside.cnm.","refresh":44,"retry":55,"expire":66}}`,
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.outside.cnm.","ns":"ns1.outside.cnm.","refresh":44,"retry":55,"expire":66}}`,
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.flatenning.cnm.","ns":"ns1.flatenning.cnm.","refresh":44,"retry":55,"expire":66},"cname_flattening":true}`,
		},
		Entries: [][][]string{
			{
				{"a",
					`{"cname":{"ttl":300, "host":"b.inside.cnm."}}`,
				},
				{"b",
					`{"cname":{"ttl":300, "host":"a.outside.cnm."}}`,
				},
			},
			{
				{"@",
					`{"a":{"ttl":300, "records":[{"ip":"127.0.0.6"}]}}`,
				},
				{"a",
					`{"cname":{"ttl":300, "host":"b.outside.cnm."}}`,
				},
				{"b",
					`{"cname":{"ttl":300, "host":"outside.cnm."}}`,
				},
			},
			{
				{"a",
					`{"cname":{"ttl":300, "host":"a.inside.cnm."}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "cname to another authoritative zone",
				Qname: "a.inside.cnm.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("a.inside.cnm. 300 IN CNAME b.inside.cnm."),
					test.CNAME("b.inside.cnm. 300 IN CNAME a.outside.cnm."),
				},
			},
			{
				Desc:  "cname to another authoritative zone with flattening",
				Qname: "a.flattening.cnm.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("a.flattening.cnm. 300 IN CNAME a.inside.cnm."),
				},
			},
		},
	},
	{
		Name:            "cname loop",
		Description:     "should properly handle cname loop",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"loop.cnm."},
		ZoneConfigs:     []string{""},
		Entries: [][][]string{
			{
				{"w",
					`{"cname":{"ttl":300, "host":"w.loop.cnm."}}`,
				},
				{"w1",
					`{"cname":{"ttl":300, "host":"w2.loop.cnm."}}`,
				},
				{"w2",
					`{"cname":{"ttl":300, "host":"w1.loop.cnm."}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "w.loop.cnm.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
			{
				Qname: "w1.loop.cnm.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
			{
				Qname: "w2.loop.cnm.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
		},
	},
	{
		Name:            "zone matching",
		Description:     "zone should match with longest prefix",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"zone.zon.", "sub1.zone.zon."},
		ZoneConfigs:     []string{"", ""},
		Entries: [][][]string{
			{
				{"sub3.sub2",
					`{"a":{"ttl":300, "records":[{"ip":"1.1.1.1"}]}}`,
				},
				{"sub1",
					`{"a":{"ttl":300, "records":[{"ip":"2.2.2.2"}]}}`,
				},
				{"sub10",
					`{"a":{"ttl":300, "records":[{"ip":"5.5.5.5"}]}}`,
				},
				{"ub1",
					`{"a":{"ttl":300, "records":[{"ip":"6.6.6.6"}]}}`,
				},
			},
			{
				{"@",
					`{"a":{"ttl":300, "records":[{"ip":"3.3.3.3"}]}}`,
				},
				{"sub2",
					`{"a":{"ttl":300, "records":[{"ip":"4.4.4.4"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "sub1.zone.zon.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("sub1.zone.zon. 300 IN A 3.3.3.3"),
				},
			},
			{
				Qname: "sub10.zone.zon.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("sub10.zone.zon. 300 IN A 5.5.5.5"),
				},
			},
			{
				Qname: "sub3.sub2.zone.zon.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("sub3.sub2.zone.zon. 300 IN A 1.1.1.1"),
				},
			},
			{
				Qname: "sub2.sub1.zone.zon.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("sub2.sub1.zone.zon. 300 IN A 4.4.4.4"),
				},
			},
			{
				Qname: "ub1.zone.zon.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ub1.zone.zon. 300 IN A 6.6.6.6"),
				},
			},
			{
				Qname: "zonee.zon.", Qtype: dns.TypeA,
				Rcode: dns.RcodeNotAuth,
			},
			{
				Qname: "azone.zon.", Qtype: dns.TypeA,
				Rcode: dns.RcodeNotAuth,
			},
		},
	},
	{
		Name:            "delegation",
		Description:     "test subdomain delegation",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"delegation.zon."},
		ZoneConfigs:     []string{""},
		Entries: [][][]string{
			{
				{"glue",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.glue.delegation.zon."},{"host":"ns2.glue.delegation.zon."}]}}`,
				},
				{"noglue",
					`{"ns":{"ttl":300, "records":[{"host":"ns1.delegated.zon."},{"host":"ns2.delegated.zon."}]}}`,
				},
				{"ns1.glue",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"ns2.glue",
					`{"a":{"ttl":300, "records":[{"ip":"5.6.7.8"}]}}`,
				},
				{"cname",
					`{"cname":{"ttl":300, "host":"glue.delegation.zon."}}`,
				},
				{"subdel",
					`{
						"ns":{"ttl":300, "records":[{"host":"ns1.delegated.zon."},{"host":"ns2.delegated.zon."}]},
						"ds":{"ttl":300, "records":[{"key_tag":57855, "algorithm":5, "digest_type":1, "digest":"B6DCD485719ADCA18E5F3D48A2331627FDD3636B"}]}
					}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "delegation with glue IPs",
				Qname: "glue.delegation.zon.",
				Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.NS("glue.delegation.zon. 300 IN NS ns1.glue.delegation.zon."),
					test.NS("glue.delegation.zon. 300 IN NS ns2.glue.delegation.zon."),
				},
				Extra: []dns.RR{
					test.A("ns1.glue.delegation.zon. 300 IN A 1.2.3.4"),
					test.A("ns2.glue.delegation.zon. 300 IN A 5.6.7.8"),
				},
			},
			{
				Desc:  "delegation without glue IPs",
				Qname: "noglue.delegation.zon.",
				Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.NS("noglue.delegation.zon. 300 IN NS ns1.delegated.zon."),
					test.NS("noglue.delegation.zon. 300 IN NS ns2.delegated.zon."),
				},
			},
			{
				Desc:  "cname to subdel",
				Qname: "cname.delegation.zon.",
				Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("cname.delegation.zon. 300 IN CNAME glue.delegation.zon."),
				},
				Ns: []dns.RR{
					test.NS("glue.delegation.zon. 300 IN NS ns1.glue.delegation.zon."),
					test.NS("glue.delegation.zon. 300 IN NS ns2.glue.delegation.zon."),
				},
				Extra: []dns.RR{
					test.A("ns1.glue.delegation.zon. 300 IN A 1.2.3.4"),
					test.A("ns2.glue.delegation.zon. 300 IN A 5.6.7.8"),
				},
			},
			{
				Desc:  "subdel with DS",
				Qname: "subdel.delegation.zon.",
				Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.DS("subdel.delegation.zon. 300 DS 57855 5 1 B6DCD485719ADCA18E5F3D48A2331627FDD3636B"),
					test.NS("subdel.delegation.zon. 300 IN NS ns1.delegated.zon."),
					test.NS("subdel.delegation.zon. 300 IN NS ns2.delegated.zon."),
				},
			},
			{
				Desc:  "delegated query",
				Qname: "x.y.subdel.delegation.zon.",
				Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.DS("subdel.delegation.zon. 300 DS 57855 5 1 B6DCD485719ADCA18E5F3D48A2331627FDD3636B"),
					test.NS("subdel.delegation.zon. 300 IN NS ns1.delegated.zon."),
					test.NS("subdel.delegation.zon. 300 IN NS ns2.delegated.zon."),
				},
			},
		},
	},
	{
		Name:            "label matching",
		Description:     "test correct label matching",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"zone1.com.", "zone2.com.", "zone3.com."},
		ZoneConfigs:     []string{"", "", ""},
		Entries: [][][]string{
			{
				{"@",
					`{"a":{"ttl":300, "records":[{"ip":"1.1.1.1"}]}}`,
				},
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"1.1.1.2"}]}}`,
				},
			},
			{
				{"@",
					`{"a":{"ttl":300, "records":[{"ip":"2.2.2.1"}]}}`,
				},
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"2.2.2.2"}]}}`,
				},
				{"zone1.com",
					`{"a":{"ttl":300, "records":[{"ip":"2.2.2.3"}]}}`,
				},
				{"www.zone1",
					`{"a":{"ttl":300, "records":[{"ip":"2.2.2.4"}]}}`,
				},
				{"www.zone1.com",
					`{"a":{"ttl":300, "records":[{"ip":"2.2.2.5"}]}}`,
				},
			},
			{
				{"@",
					`{"a":{"ttl":300, "records":[{"ip":"3.3.3.1"}]}}`,
				},
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"3.3.3.2"}]}}`,
				},
				{"zone3.com",
					`{"a":{"ttl":300, "records":[{"ip":"3.3.3.3"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "zone1.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("zone1.com. 300 IN A 1.1.1.1"),
				},
			},
			{
				Qname: "www.zone1.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("www.zone1.com. 300 IN A 1.1.1.2"),
				},
			},
			{
				Qname: "zone2.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("zone2.com. 300 IN A 2.2.2.1"),
				},
			},
			{
				Qname: "www.zone2.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("www.zone2.com. 300 IN A 2.2.2.2"),
				},
			},
			{
				Qname: "zone1.com.zone2.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("zone1.com.zone2.com. 300 IN A 2.2.2.3"),
				},
			},
			{
				Qname: "www.zone1.zone2.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("www.zone1.zone2.com. 300 IN A 2.2.2.4"),
				},
			},
			{
				Qname: "www.zone1.com.zone2.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("www.zone1.com.zone2.com. 300 IN A 2.2.2.5"),
				},
			},
			{
				Qname: "zone3.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("zone3.com. 300 IN A 3.3.3.1"),
				},
			},
			{
				Qname: "www.zone3.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("www.zone3.com. 300 IN A 3.3.3.2"),
				},
			},
			{
				Qname: "zone3.com.zone3.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("zone3.com.zone3.com. 300 IN A 3.3.3.3"),
				},
			},
		},
	},
	{
		Name:            "cname flattening leaving zone",
		Description:     "test correct response when reaching a cname pointing outside current zone",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"flat.com.", "noflat.com."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.flat.com.","ns":"ns1.flat.com.","refresh":44,"retry":55,"expire":66},"cname_flattening":true}}`,
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.noflat.com.","ns":"ns1.noflat.com.","refresh":44,"retry":55,"expire":66},,"cname_flattening":false}}`,
		},
		Entries: [][][]string{
			{
				{"a",
					`{"cname":{"ttl":300, "host":"www.flat.com."}}`,
				},
				{"www",
					`{"cname":{"ttl":300, "host":"anotherzone.com."}}`,
				},
			},
			{
				{"a",
					`{"cname":{"ttl":300, "host":"www.noflat.com."}}`,
				},
				{"www",
					`{"cname":{"ttl":300, "host":"anotherzone.com."}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "a.flat.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("a.flat.com. 300 IN CNAME anotherzone.com."),
				},
			},
			{
				Qname: "a.noflat.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("a.noflat.com. 300 IN CNAME www.noflat.com."),
					test.CNAME("www.noflat.com. 300 IN CNAME anotherzone.com."),
				},
			},
		},
	},
	{
		Name:            "ANAME ttl",
		Description:     "test ttl value for aname queries",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.com."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.com.","ns":"ns1.example.com.","refresh":44,"retry":55,"expire":66}}`,
		},
		Entries: [][][]string{
			{
				{"@",
					`{"aname":{"location":"aname.example.com."}}`,
				},
				{"upstream",
					`{"aname":{"location":"dns.msftncsi.com."}}`,
				},
				{"aname",
					`{"a":{"ttl":180, "records":[{"ip":"6.5.6.5"}]}, "aaaa":{"ttl":300, "records":[{"ip":"::1"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "example.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("example.com. 180 IN A 6.5.6.5"),
				},
			},
			{
				Qname: "upstream.example.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("upstream.example.com. 303 IN A 131.107.255.255"),
				},
			},
		},
	},
	{
		Name:            "malformed data",
		Description:     "test proper handling of malformed data",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize: func(testCase *TestCase) (*DnsRequestHandler, error) {
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
				if err := r.EnableLocation(zone, "www"); err != nil {
					return nil, errors.New(fmt.Sprintf("cannot enable location : www, %s", err))
				}
				if err := r.EnableLocation(zone, "mal1"); err != nil {
					return nil, errors.New(fmt.Sprintf("cannot enable location : mal1, %s", err))
				}
				if err := r.SetRRSetFromJson(zone, testCase.Entries[0][0][0], types.TypeANAME, testCase.Entries[0][0][1]); err != nil {
					return nil, errors.New(fmt.Sprintf("[ERROR] SetRRSetFromJson: %s\n%s", err, testCase.Entries[0][0][1]))
				}
				if err := r.SetRRSetFromJson(zone, testCase.Entries[0][1][0], types.TypeANAME, testCase.Entries[0][1][1]); err != nil {
					return nil, errors.New(fmt.Sprintf("[ERROR] SetRRSetFromJson: %s\n%s", err, testCase.Entries[0][1][1]))
				}
				if err := r.SetRRSetFromJson(zone, testCase.Entries[0][2][0], types.TypeANAME, testCase.Entries[0][2][1]); err != nil {
					return nil, errors.New(fmt.Sprintf("[ERROR] SetRRSetFromJson: %s\n%s", err, testCase.Entries[0][2][1]))
				}
				if err := r.SetZoneConfigFromJson(zone, testCase.ZoneConfigs[i]); err != nil {
					return nil, err
				}
			}
			h.RedisData.LoadZones()
			return h, nil
		},
		ApplyAndVerify: DefaultApplyAndVerify,
		Zones:          []string{"example.mal."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.mal.","ns":"ns1.example.mal.","refresh":44,"retry":55,"expire":66}}`,
		},
		Entries: [][][]string{
			{
				{"@",
					`{"location":"mal1.example.mal."}`,
				},
				{"www",
					`{"ttl":"300", "records":[{"ip":"3.3.3.1"}]}`,
				},
				{"mal1",
					`"!@#$$^$*^&^dfgsfdg@#@EWDS"`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  " aname to garbage",
				Qname: "example.mal.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
			{
				Desc:  "invalid json format",
				Qname: "www.example.mal.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
			{
				Desc:  "garbage",
				Qname: "mal1.example.mal.", Qtype: dns.TypeA,
				Rcode: dns.RcodeServerFailure,
			},
		},
	},
	{
		Name:            "implicit root location",
		Description:     "root location always exists",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"example.root."},
		ZoneConfigs: []string{
			`{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.example.root.","ns":"ns1.example.root.","refresh":44,"retry":55,"expire":66}}`,
		},
		Entries: [][][]string{
			{
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"3.3.3.1"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "example.root.", Qtype: dns.TypeA,
				Ns: []dns.RR{
					test.SOA("example.root. 300 IN SOA ns1.example.root. hostmaster.example.root. 1460498836 44 55 66 100"),
				},
			},
			{
				Qname: "example.root.", Qtype: dns.TypeSOA,
				Rcode: dns.RcodeSuccess,
				Answer: []dns.RR{
					test.SOA("example.root. 300 IN SOA ns1.example.root. hostmaster.example.root. 1460498836 44 55 66 100"),
				},
			},
			{
				Qname: "example.root.", Qtype: dns.TypeTXT,
				Ns: []dns.RR{
					test.SOA("example.root. 300 IN SOA ns1.example.root. hostmaster.example.root. 1460498836 44 55 66 100"),
				},
			},
		},
	},
	{
		Name:            "cache stale",
		Description:     "use stale data from cache when redis is not available",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize: func(testCase *TestCase) (handler *DnsRequestHandler, e error) {
			testCase.RedisDataConfig.Redis.Connection.WaitForConnection = false
			testCase.RedisDataConfig.Redis.Connection.MaxActiveConnections = 0
			testCase.RedisDataConfig.Redis.Connection.MaxIdleConnections = 0
			testCase.RedisDataConfig.RecordCacheTimeout = 1
			return DefaultInitialize(testCase)
		},
		ApplyAndVerify: func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
			RegisterTestingT(t)
			tc := testCase.TestCases[0]
			r := tc.Msg()
			w := test.NewRecorder(&test.ResponseWriter{})
			state := NewRequestContext(w, r)
			handler.HandleRequest(state)

			resp := w.Msg

			err := test.SortAndCheck(resp, tc)
			Expect(err).To(BeNil())

			time.Sleep(time.Duration(1200) * time.Millisecond)

			r = tc.Msg()
			w = test.NewRecorder(&test.ResponseWriter{})
			state = NewRequestContext(w, r)
			handler.HandleRequest(state)

			resp = w.Msg
			err = test.SortAndCheck(resp, tc)
			Expect(err).To(BeNil())
		},
		Zones:       []string{"stale.com."},
		ZoneConfigs: []string{""},
		Entries: [][][]string{
			{
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"3.3.3.1"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "www.stale.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("www.stale.com. 300 IN A 3.3.3.1"),
				},
			},
		},
	},
	{
		Name:            "zone list update",
		Description:     "test zone list update",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize: func(testCase *TestCase) (handler *DnsRequestHandler, e error) {
			testCase.RedisDataConfig.ZoneReload = 1
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
						return nil, errors.New(fmt.Sprintf("[ERROR] 3: %s\n%s", err, cmd[1]))
					}
				}
				if err := r.SetZoneConfigFromJson(zone, testCase.ZoneConfigs[i]); err != nil {
					return nil, err
				}
			}
			time.Sleep(time.Second)
			return h, nil
		},
		ApplyAndVerify: func(testCase *TestCase, handler *DnsRequestHandler, t *testing.T) {
			dh := storage.NewDataHandler(&testCase.RedisDataConfig)
			RegisterTestingT(t)
			{
				_ = dh.DisableZone(testCase.Zones[0])
				time.Sleep(time.Millisecond * 1200)

				tc := testCase.TestCases[0]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg

				err := test.SortAndCheck(resp, tc)
				Expect(err).To(BeNil())
			}

			{
				_ = dh.EnableZone(testCase.Zones[0])
				time.Sleep(time.Millisecond * 1200)

				tc := testCase.TestCases[1]
				r := tc.Msg()
				w := test.NewRecorder(&test.ResponseWriter{})
				state := NewRequestContext(w, r)
				handler.HandleRequest(state)

				resp := w.Msg

				err := test.SortAndCheck(resp, tc)
				Expect(err).To(BeNil())
			}
		},
		Zones:       []string{"zone1.zon.", "zone2.zon."},
		ZoneConfigs: []string{"", ""},
		Entries: [][][]string{
			{
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
			},
			{
				{"www",
					`{"a":{"ttl":300, "records":[{"ip":"2.3.4.5"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Qname: "www.zone1.zon", Qtype: dns.TypeA,
				Rcode: dns.RcodeNotAuth,
			},
			{
				Qname: "www.zone1.zon.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("www.zone1.zon. 300 IN A 1.2.3.4"),
				},
			},
		},
	},
	{
		Name:            "IDN zones",
		Description:     "test zone names with IDN values (internationalized domain names)",
		Enabled:         true,
		RedisDataConfig: DefaultRedisDataTestConfig,
		HandlerConfig:   DefaultHandlerTestConfig,
		Initialize:      DefaultInitialize,
		ApplyAndVerify:  DefaultApplyAndVerify,
		Zones:           []string{"ουτοπία.δπθ.gr.", "ascii.com."},
		ZoneConfigs:     []string{"", ""},
		Entries: [][][]string{
			{
				{"@",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
				{"ουτοπία",
					`{"a":{"ttl":300, "records":[{"ip":"2.3.4.5"}]}}`,
				},
			},
			{
				{"@",
					`{"aname":{"location":"ουτοπία.ascii.com."}}`,
				},
				{"www",
					`{"cname":{"ttl":300, "host":"ουτοπία.δπθ.gr."}}`,
				},
				{"ουτοπία",
					`{"a":{"ttl":300, "records":[{"ip":"1.2.3.4"}]}}`,
				},
			},
		},
		TestCases: []test.Case{
			{
				Desc:  "query to IDN root",
				Qname: "ουτοπία.δπθ.gr.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ουτοπία.δπθ.gr. 300 IN A 1.2.3.4"),
				},
			},
			{
				Desc:  "query to IDN subdomain",
				Qname: "ουτοπία.ουτοπία.δπθ.gr.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ουτοπία.ουτοπία.δπθ.gr. 300 IN A 2.3.4.5"),
				},
			},
			{
				Qname: "ascii.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ascii.com. 300 IN A 1.2.3.4"),
				},
			},
			{
				Desc:  "cname to IDN",
				Qname: "www.ascii.com.", Qtype: dns.TypeCNAME,
				Answer: []dns.RR{
					test.CNAME("www.ascii.com. 300 IN CNAME ουτοπία.δπθ.gr."),
				},
			},
			{
				Desc:  "ascii zone with IDN subdomain",
				Qname: "ουτοπία.ascii.com.", Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ουτοπία.ascii.com. 300 IN A 1.2.3.4"),
				},
			},
		},
	},
}

func TestAllHandler(t *testing.T) {
	RegisterTestingT(t)
	for _, testCase := range handlerTestCases {
		if testCase.Name != "Basic Usage" {
			continue
		}
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

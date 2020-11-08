package redis

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/hawell/z42/types"
	"github.com/miekg/dns"
)

var dataHandlerDefaultTestConfig = DataHandlerConfig{
	ZoneCacheSize:      10000,
	ZoneCacheTimeout:   60,
	ZoneReload:         1,
	RecordCacheSize:    1000000,
	RecordCacheTimeout: 60,
	Redis: RedisConfig{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
		Connection: RedisConnectionConfig{
			MaxIdleConnections:   10,
			MaxActiveConnections: 10,
			ConnectTimeout:       600,
			ReadTimeout:          600,
			IdleKeepAlive:        6000,
			MaxKeepAlive:         6000,
			WaitForConnection:    true,
		},
	},
}

func TestGetLocation(t *testing.T) {
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	_ = dh.Clear()
	_ = dh.EnableZone("zone1.com.")
	_ = dh.SetLocationFromJson("zone1.com.", "@", `{"a":{"ttl":300, "records":[{"ip":"5.5.5.5"}]}}`)
	location := types.Record{
		RRSets: types.RRSets{
			A: types.IP_RRSet{
				FilterConfig: types.IpFilterConfig{
					Count:     "multi",
					Order:     "none",
					GeoFilter: "none",
				},
				Ttl: 300,
				Data: []types.IP_RR{
					{Ip: net.ParseIP("5.5.5.5")},
				},
			},
		},
		Label: "@",
		Fqdn:  "zone1.com.",
	}
	l, err := dh.GetLocation("zone1.com.", "@")
	if err != nil {
		t.Fail()
	}
	if l == nil {
		t.Fail()
	}
	if l.Fqdn != location.Fqdn {
		fmt.Println("fqdn name mismatch", l.Fqdn, location.Fqdn)
		t.Fail()
	}
	if l.Label != location.Label {
		fmt.Println("label name mismatch", l.Label, location.Label)
		t.Fail()
	}
	if reflect.DeepEqual(l.A, location.A) == false {
		fmt.Println(l.A)
		fmt.Println(location.A)
		t.Fail()
	}
}

func TestSetLocation(t *testing.T) {
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	_ = dh.Clear()
	_ = dh.EnableZone("zone1.com.")
	location := types.Record{
		RRSets: types.RRSets{
			A: types.IP_RRSet{
				FilterConfig: types.IpFilterConfig{
					Count:     "multi",
					Order:     "none",
					GeoFilter: "none",
				},
				Ttl: 300,
				Data: []types.IP_RR{
					{Ip: net.ParseIP("5.5.5.5")},
				},
			},
		},
		Label: "@",
		Fqdn:  "zone1.com.",
	}
	err := dh.SetLocation("zone1.com.", "@", &location)
	if err != nil {
		t.Fail()
	}
	l, err := dh.GetLocation("zone1.com.", "@")
	if err != nil {
		t.Fail()
	}
	if l.Fqdn != location.Fqdn {
		fmt.Println("fqdn name mismatch", l.Fqdn, location.Fqdn)
		t.Fail()
	}
	if l.Label != location.Label {
		fmt.Println("label name mismatch", l.Label, location.Label)
		t.Fail()
	}
	if reflect.DeepEqual(l.A, location.A) == false {
		fmt.Println("l.A not equal location.A", l.A, location.A)
		t.Fail()
	}
}

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
zone1.com. IN DNSKEY 256 3 5 AwEAAaKsF5vxBfKuqeUa4+ugW37ftFZOyo+k7r2aeJzZdIbYk//P/dpC HK4uYG8Z1dr/qeo12ECNVcf76j+XAdJD841ELiRVaZteH8TqfPQ+jdHz 10e8Sfkh7OZ4oBwSCXWj+Q==
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
zone1.com. IN DNSKEY 257 3 5 AwEAAeVrjiD9xhyA+UJnnei/tnoQQpLrEwFzb/blH6c80yR7APmwXrGU hbETczAFdnazO3wKXC+SIDaq4W+bcMbtf/nGY9i3dzwC25BDc5/3q05e AOLkHUlnZI/Cp2i4iUD2kw==
`

func TestGetZone(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	_ = dh.Clear()
	_ = dh.EnableZone("zone1.com")
	_ = dh.SetZoneConfigFromJson("zone1.com.", `{"domain_id":"123456", "soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.zone1.com.","ns":"ns1.zone1.com.","refresh":44,"retry":55,"expire":66, "serial":32343}, "dnssec": true}`)
	_ = dh.SetZoneKey(zoneName, "zsk", zone1ZskPub, zone1ZskPriv)
	_ = dh.SetZoneKey(zoneName, "ksk", zone1KskPub, zone1KskPriv)

	zone := dh.GetZone(zoneName)
	if zone == nil {
		fmt.Println("load zone failed")
		t.Fail()
		return
	}
	if zone.Name != zoneName {
		fmt.Println("zone name mismatch")
		t.Fail()
	}
	zoneConfig := types.ZoneConfig{
		DomainId: "123456",
		SOA: &types.SOA_RRSet{
			Data: &dns.SOA{
				Hdr: dns.RR_Header{
					Name:     "zone1.com.",
					Rrtype:   6,
					Class:    1,
					Ttl:      300,
					Rdlength: 0,
				},
				Ns:      "ns1.zone1.com.",
				Mbox:    "hostmaster.zone1.com.",
				Serial:  32343,
				Refresh: 44,
				Retry:   55,
				Expire:  66,
				Minttl:  100,
			},
			Ns:      "ns1.zone1.com.",
			MBox:    "hostmaster.zone1.com.",
			Ttl:     300,
			Refresh: 44,
			Retry:   55,
			Expire:  66,
			MinTtl:  100,
			Serial:  32343,
		},
		DnsSec:          true,
		CnameFlattening: false,
	}
	if cmp.Equal(zone.Config, &zoneConfig) == false {
		fmt.Println(cmp.Diff(zone.Config, zoneConfig))
		fmt.Println("config mismatch")
		t.Fail()
	}
}

func TestEnableZone(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	dh.Clear()
	dh.EnableZone(zoneName)
	zone := dh.GetZone(zoneName)
	if zone == nil {
		t.Fail()
	}
}

func TestDisableZone(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	dh.Clear()
	dh.EnableZone(zoneName)
	dh.DisableZone(zoneName)
	time.Sleep(time.Second * 2)
	zone := dh.FindZone(zoneName)
	if zone != "" {
		t.Fail()
	}
}

func TestSetZoneConfig(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	dh.Clear()
	dh.EnableZone(zoneName)
	zone := dh.GetZone(zoneName)
	if zone == nil {
		t.Fail()
	}
	zone.Config.DomainId = "12345"
	dh.SetZoneConfig(zoneName, zone.Config)
	time.Sleep(time.Second * 2)
	zone = dh.GetZone(zoneName)
	if zone.Config.DomainId != "12345" {
		t.Fail()
	}
}

func TestSetZoneKey(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	dh.Clear()
	dh.EnableZone(zoneName)
	zoneConfig := types.ZoneConfig{
		DomainId: "123456",
		SOA: &types.SOA_RRSet{
			Data: &dns.SOA{
				Hdr: dns.RR_Header{
					Name:     "zone1.com.",
					Rrtype:   6,
					Class:    1,
					Ttl:      300,
					Rdlength: 0,
				},
				Ns:      "ns1.zone1.com.",
				Mbox:    "hostmaster.zone1.com.",
				Serial:  32343,
				Refresh: 44,
				Retry:   55,
				Expire:  66,
				Minttl:  100,
			},
			Ns:      "ns1.zone1.com.",
			MBox:    "hostmaster.zone1.com.",
			Ttl:     300,
			Refresh: 44,
			Retry:   55,
			Expire:  66,
			MinTtl:  100,
			Serial:  32343,
		},
		DnsSec:          false,
		CnameFlattening: false,
	}
	dh.SetZoneConfig(zoneName, &zoneConfig)
	dh.SetZoneKey(zoneName, "zsk", zone1ZskPub, zone1ZskPriv)
	dh.SetZoneKey(zoneName, "ksk", zone1KskPub, zone1KskPriv)
	zoneConfig.DnsSec = true
	dh.SetZoneConfig(zoneName, &zoneConfig)
	zone := dh.GetZone(zoneName)
	if zone == nil {
		t.Fail()
	}
	if zone.Config.DnsSec == false {
		t.Fail()
	}
}

func TestLoadZones(t *testing.T) {
}

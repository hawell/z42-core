package redis

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	jsoniter "github.com/json-iterator/go"
	"net"
	"reflect"
	"sort"
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

func TestEnableZone(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.Fail()
	}
	zone := dh.GetZone(zoneName)
	if zone == nil {
		t.FailNow()
	}
	if zone.Name != zoneName {
		t.Fail()
	}
}

func TestDisableZone(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.Fail()
	}
	zone := dh.FindZone(zoneName)
	if zone == "" {
		t.FailNow()
	}
	if zone != zoneName {
		t.Fail()
	}
	err = dh.DisableZone(zoneName)
	if err != nil {
		t.Fail()
	}
	time.Sleep(time.Millisecond * 1200)
	zone = dh.FindZone(zoneName)
	if zone != "" {
		fmt.Println(dh.GetZones())
		t.Fail()
	}
}

func TestFindZone(t *testing.T) {
	zone1Name := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	z := dh.FindZone(zone1Name)
	if z != "" {
		t.Fail()
	}
	err = dh.EnableZone(zone1Name)
	if err != nil {
		t.Fail()
	}
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone(zone1Name)
	if z != zone1Name {
		t.Fail()
	}
	z = dh.FindZone("a.b.c.d." + zone1Name)
	if z != zone1Name {
		t.Fail()
	}
	subZoneName := "sub.zone1.com."
	err = dh.EnableZone(subZoneName)
	if err != nil {
		t.Fail()
	}
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone("a.b.c." + subZoneName)
	if z != subZoneName {
		t.Fail()
	}
	err = dh.DisableZone(subZoneName)
	if err != nil {
		t.Fail()
	}
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone("a.b.c." + subZoneName)
	if z != zone1Name {
		t.Fail()
	}
	err = dh.DisableZone(zone1Name)
	if err != nil {
		t.Fail()
	}
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone("a.b.c." + subZoneName)
	if z != "" {
		t.Fail()
	}
}

func TestGetZone(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.Fail()
	}
	zone := dh.GetZone(zoneName)
	if zone == nil {
		fmt.Println("load zone failed")
		t.FailNow()
	}
	if zone.Name != zoneName {
		fmt.Println("zone name mismatch")
		t.Fail()
	}
	defaultConfig := types.ZoneConfigFromJson(zoneName, "")
	if cmp.Equal(zone.Config, defaultConfig) == false {
		fmt.Println(cmp.Diff(zone.Config, defaultConfig))
		fmt.Println("config mismatch")
		t.Fail()
	}
}

func TestGetZoneConfig(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.Fail()
	}
	defaultConfig := types.ZoneConfigFromJson(zoneName, "")
	zoneConfig, err := dh.GetZoneConfig(zoneName)
	if err != nil {
		t.Fail()
	}
	if cmp.Equal(defaultConfig, zoneConfig) == false {
		t.Fail()
	}
}

func TestGetZones(t *testing.T) {
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	zones := []string{"zone1.com.", "zone2.com.", "zone3.com.", "zone4.com.", "zone5.com."}
	for _, z := range zones {
		err = dh.EnableZone(z)
		if err != nil {
			t.Fail()
		}
	}
	time.Sleep(time.Millisecond * 200)
	recvdZones := dh.GetZones()
	sort.Strings(recvdZones)
	if !cmp.Equal(zones, recvdZones) {
		t.Fail()
	}
}

func TestGetZoneLocations(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.Fail()
	}
	locations := dh.GetZoneLocations(zoneName)
	if len(locations) != 0 {
		t.Fail()
	}
}

func TestSetZoneConfig(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.FailNow()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.FailNow()
	}
	oldZoneConfig, err := dh.GetZoneConfig(zoneName)
	if err != nil {
		t.FailNow()
	}
	oldZoneConfig.DomainId = "12345"
	err = dh.SetZoneConfig(zoneName, oldZoneConfig)
	if err != nil {
		t.FailNow()
	}
	time.Sleep(time.Second * 2)
	newZoneConfig, err := dh.GetZoneConfig(zoneName)
	if err != nil {
		t.Fail()
	}
	if cmp.Equal(oldZoneConfig, newZoneConfig) == false {
		t.FailNow()
	}
}

func TestSetZoneConfigFromJson(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.FailNow()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.FailNow()
	}
	configStr := `{"soa":{"ttl":311, "minttl":111, "mbox":"hostmaster.example.root.","ns":"ns1.example.root.","refresh":44,"retry":55,"expire":66}, "cname_flattening":true, "domain_id":"12345"}`
	err = dh.SetZoneConfigFromJson(zoneName, configStr)
	if err != nil {
		t.Fail()
	}
	config, err := dh.GetZoneConfig(zoneName)
	if err != nil {
		t.FailNow()
	}
	if config.SOA.Ttl != 311 {
		t.Fail()
	}
	if config.SOA.MinTtl != 111 {
		t.Fail()
	}
	if config.SOA.MBox != "hostmaster.example.root." {
		t.Fail()
	}
	if config.SOA.Ns != "ns1.example.root." {
		t.Fail()
	}
	if config.SOA.Refresh != 44 || config.SOA.Retry != 55 || config.SOA.Expire != 66 {
		t.Fail()
	}
	if config.CnameFlattening != true {
		t.Fail()
	}
	if config.DomainId != "12345" {
		t.Fail()
	}
}

func TestGetLocation(t *testing.T) {
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.FailNow()
	}
	err = dh.EnableZone("zone1.com.")
	if err != nil {
		t.Fail()
	}
	err = dh.SetLocationFromJson("zone1.com.", "@", `{"a":{"ttl":300, "records":[{"ip":"5.5.5.5"}]}}`)
	if err != nil {
		t.Fail()
	}
	location := types.Record{
		RRSets: types.RRSets{
			A: types.IP_RRSet{
				FilterConfig: types.IpFilterConfig{
					Count:     "",
					Order:     "",
					GeoFilter: "",
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
		t.FailNow()
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
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	zoneName := "zone1.com."
	err = dh.EnableZone(zoneName)
	if err != nil {
		t.Fail()
	}
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
		Fqdn:  zoneName,
	}
	err = dh.SetLocation(zoneName, "@", &location)
	if err != nil {
		t.Fail()
	}
	l, err := dh.GetLocation(zoneName, "@")
	if err != nil {
		t.FailNow()
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

func TestSetLocationFromJson(t *testing.T) {
	zoneName := "example.com."
	locationStr :=
		`{
			"a":{"ttl":300, "records":[{"ip":"1.2.3.4", "country":["ES"]},{"ip":"5.6.7.8", "country":[""]}]},
			"aaaa":{"ttl":300, "records":[{"ip":"::1"}]},
			"cname":{"ttl":300, "host":"x.example.com."},
			"txt":{"ttl":300, "records":[{"text":"foo"},{"text":"bar"}]},
			"ns":{"ttl":300, "records":[{"host":"ns1.example.com."},{"host":"ns2.example.com."}]},
			"mx":{"ttl":300, "records":[{"host":"mx1.example.com.", "preference":10},{"host":"mx2.example.com.", "preference":10}]},
			"srv":{"ttl":300, "records":[{"target":"sip.example.com.","port":555,"priority":10,"weight":100}]},
			"tlsa":{"ttl":300, "records":[{"usage":0, "selector":0, "matching_type":1, "certificate":"d2abde240d7cd3ee6b4b28c54df034b97983a1d16e8a410e4561cb106618e971"}]},
			"ds":{"ttl":300, "records":[{"key_tag":57855, "algorithm":5, "digest_type":1, "digest":"B6DCD485719ADCA18E5F3D48A2331627FDD3636B"}]},
			"aname":{"location":"aname.example.com."},
			"caa":{"ttl":300, "records":[{"tag":"issue", "value":"godaddy2.com;", "flag":0}]}
		}`
	location := types.Record{
		Label:        "@",
		Fqdn:         "example.com.",
		CacheTimeout: time.Now().Unix() + int64(dataHandlerDefaultTestConfig.RecordCacheTimeout),
	}
	err := jsoniter.Unmarshal([]byte(locationStr), &location)
	if err != nil {
		fmt.Println("1", err)
		t.Fail()
	}
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err = dh.Clear()
	if err != nil {
		fmt.Println("2")
		t.Fail()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
		fmt.Println("3")
		t.Fail()
	}
	err = dh.SetLocationFromJson(zoneName, "@", locationStr)
	if err != nil {
		fmt.Println("4")
		t.Fail()
	}
	l, err := dh.GetLocation(zoneName, "@")
	if err != nil {
		fmt.Println("5")
		t.FailNow()
	}
	if l.Fqdn != zoneName {
		fmt.Println("6")
		t.Fail()
	}
	if l.Label != "@" {
		fmt.Println("7")
		t.Fail()
	}
	if cmp.Equal(&location, l) != true {
		fmt.Println(cmp.Diff(&location, l))
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

func TestSetZoneKey(t *testing.T) {
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	if err != nil {
		t.Fail()
	}
	err = dh.EnableZone(zoneName)
	if err != nil {
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
		DnsSec:          false,
		CnameFlattening: false,
	}
	err = dh.SetZoneConfig(zoneName, &zoneConfig)
	if err != nil {
		t.Fail()
	}
	err = dh.SetZoneKey(zoneName, "zsk", zone1ZskPub, zone1ZskPriv)
	if err != nil {
		t.Fail()
	}
	err = dh.SetZoneKey(zoneName, "ksk", zone1KskPub, zone1KskPriv)
	if err != nil {
		t.Fail()
	}
	zoneConfig.DnsSec = true
	err = dh.SetZoneConfig(zoneName, &zoneConfig)
	if err != nil {
		t.Fail()
	}
	zone := dh.GetZone(zoneName)
	if zone == nil {
		t.FailNow()
	}
	if zone.Config.DnsSec == false {
		t.Fail()
	}
}

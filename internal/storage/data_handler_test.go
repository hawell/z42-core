package storage

import (
	redisCon "github.com/gomodule/redigo/redis"
	"github.com/google/go-cmp/cmp"
	"github.com/hawell/z42/pkg/hiredis"
	jsoniter "github.com/json-iterator/go"
	. "github.com/onsi/gomega"
	"net"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/hawell/z42/internal/types"
	"github.com/miekg/dns"
)

var dataHandlerDefaultTestConfig = DataHandlerConfig{
	ZoneCacheSize:      10000,
	ZoneCacheTimeout:   60,
	ZoneReload:         1,
	RecordCacheSize:    1000000,
	RecordCacheTimeout: 60,
	Redis: hiredis.Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
		Connection: hiredis.ConnectionConfig{
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
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	zone := dh.GetZone(zoneName)
	g.Expect(zone).NotTo(BeNil())
	g.Expect(zone.Name).To(Equal(zoneName))
}

func TestDisableZone(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	zone := dh.FindZone(zoneName)
	g.Expect(zone).NotTo(BeEmpty())
	g.Expect(zone).To(Equal(zoneName))
	err = dh.DisableZone(zoneName)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	zone = dh.FindZone(zoneName)
	g.Expect(zone).To(BeEmpty())
}

func TestFindZone(t *testing.T) {
	g := NewGomegaWithT(t)
	zone1Name := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	z := dh.FindZone(zone1Name)
	g.Expect(z).To(BeEmpty())
	err = dh.EnableZone(zone1Name)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone(zone1Name)
	g.Expect(z).To(Equal(zone1Name))
	z = dh.FindZone("a.b.c.d." + zone1Name)
	g.Expect(z).To(Equal(zone1Name))
	subZoneName := "sub.zone1.com."
	err = dh.EnableZone(subZoneName)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone("a.b.c." + subZoneName)
	g.Expect(z).To(Equal(subZoneName))
	err = dh.DisableZone(subZoneName)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone("a.b.c." + subZoneName)
	g.Expect(z).To(Equal(zone1Name))
	err = dh.DisableZone(zone1Name)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	z = dh.FindZone("a.b.c." + subZoneName)
	g.Expect(z).To(BeEmpty())
}

func TestGetZone(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	zone := dh.GetZone(zoneName)
	g.Expect(zone).NotTo(BeNil())
	g.Expect(zone.Name).To(Equal(zoneName))
	defaultConfig := types.ZoneConfigFromJson(zoneName, "")
	g.Expect(zone.Config).To(Equal(defaultConfig))
}

func TestGetZoneConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	defaultConfig := types.ZoneConfigFromJson(zoneName, "")
	zoneConfig, err := dh.GetZoneConfig(zoneName)
	g.Expect(err).To(BeNil())
	g.Expect(zoneConfig).To(Equal(defaultConfig))
}

func TestGetZones(t *testing.T) {
	g := NewGomegaWithT(t)
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	zones := []string{"zone1.com.", "zone2.com.", "zone3.com.", "zone4.com.", "zone5.com."}
	for _, z := range zones {
		err = dh.EnableZone(z)
		g.Expect(err).To(BeNil())
	}
	time.Sleep(time.Millisecond * 200)
	recvdZones := dh.GetZones()
	sort.Strings(recvdZones)
	g.Expect(zones).To(Equal(recvdZones))
}

func TestGetZoneLocations(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	locations := dh.GetZoneLocations(zoneName)
	g.Expect(locations).To(BeEmpty())
}

func TestSetZoneConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	oldZoneConfig, err := dh.GetZoneConfig(zoneName)
	g.Expect(err).To(BeNil())
	oldZoneConfig.DomainId = "12345"
	err = dh.SetZoneConfig(zoneName, oldZoneConfig)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Second * 2)
	newZoneConfig, err := dh.GetZoneConfig(zoneName)
	g.Expect(err).To(BeNil())
	g.Expect(cmp.Equal(oldZoneConfig, newZoneConfig)).To(BeTrue())
}

func TestSetZoneConfigFromJson(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	configStr := `{"soa":{"ttl":311, "minttl":111, "mbox":"hostmaster.example.root.","ns":"ns1.example.root.","refresh":44,"retry":55,"expire":66}, "cname_flattening":true, "domain_id":"12345"}`
	err = dh.SetZoneConfigFromJson(zoneName, configStr)
	g.Expect(err).To(BeNil())
	config, err := dh.GetZoneConfig(zoneName)
	g.Expect(err).To(BeNil())
	g.Expect(config.SOA.Ttl).To(Equal(uint32(311)))
	g.Expect(config.SOA.MinTtl).To(Equal(uint32(111)))
	g.Expect(config.SOA.MBox).To(Equal("hostmaster.example.root."))
	g.Expect(config.SOA.Ns).To(Equal("ns1.example.root."))
	g.Expect(config.SOA.Refresh).To(Equal(uint32(44)))
	g.Expect(config.SOA.Retry).To(Equal(uint32(55)))
	g.Expect(config.SOA.Expire).To(Equal(uint32(66)))
	g.Expect(config.CnameFlattening).To(BeTrue())
	g.Expect(config.DomainId).To(Equal("12345"))
}

func TestGetLocation(t *testing.T) {
	g := NewGomegaWithT(t)
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone("zone1.com.")
	g.Expect(err).To(BeNil())
	err = dh.SetLocationFromJson("zone1.com.", "@", `{"a":{"ttl":300, "records":[{"ip":"5.5.5.5"}]}}`)
	g.Expect(err).To(BeNil())
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
	}
	l, err := dh.GetLocation("zone1.com.", "@")
	g.Expect(err).To(BeNil())
	g.Expect(l).NotTo(BeNil())
	g.Expect(reflect.DeepEqual(l.A, location.A)).To(BeTrue())
}

func TestSetLocation(t *testing.T) {
	g := NewGomegaWithT(t)
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	zoneName := "zone1.com."
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
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
	}
	err = dh.SetLocation(zoneName, "@", &location)
	g.Expect(err).To(BeNil())
	l, err := dh.GetLocation(zoneName, "@")
	g.Expect(err).To(BeNil())
	g.Expect(reflect.DeepEqual(l.A, location.A)).To(BeTrue())
}

func TestSetLocationFromJson(t *testing.T) {
	g := NewGomegaWithT(t)
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
		CacheTimeout: time.Now().Unix() + int64(dataHandlerDefaultTestConfig.RecordCacheTimeout),
	}
	err := jsoniter.Unmarshal([]byte(locationStr), &location)
	g.Expect(err).To(BeNil())
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err = dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	err = dh.SetLocationFromJson(zoneName, "@", locationStr)
	g.Expect(err).To(BeNil())
	l, err := dh.GetLocation(zoneName, "@")
	g.Expect(err).To(BeNil())
	g.Expect(cmp.Equal(&location, l)).To(BeTrue())
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
	g := NewGomegaWithT(t)
	zoneName := "zone1.com."
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
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
	g.Expect(err).To(BeNil())
	err = dh.SetZoneKey(zoneName, "zsk", zone1ZskPub, zone1ZskPriv)
	g.Expect(err).To(BeNil())
	err = dh.SetZoneKey(zoneName, "ksk", zone1KskPub, zone1KskPriv)
	g.Expect(err).To(BeNil())
	zoneConfig.DnsSec = true
	err = dh.SetZoneConfig(zoneName, &zoneConfig)
	g.Expect(err).To(BeNil())
	zone := dh.GetZone(zoneName)
	g.Expect(zone).NotTo(BeNil())
	g.Expect(zone.Config.DnsSec).To(BeTrue())
}

func TestLocationUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "example.com."
	locationStr := `{"a":{"ttl":300, "records":[{"ip":"1.2.3.4", "country":["ES"]},{"ip":"5.6.7.8", "country":[""]}]}}`
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	_, err = dh.GetLocation(zoneName, "@")
	g.Expect(err).To(BeNil())

	err = dh.SetLocationFromJson(zoneName, "@", locationStr)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	location, err := dh.GetLocation(zoneName, "@")
	g.Expect(err).To(BeNil())
	g.Expect(len(location.A.Data)).To(Equal(2))
	g.Expect(location.A.Data[0].Ip.String()).To(Equal("1.2.3.4"))
	g.Expect(location.A.Data[1].Ip.String()).To(Equal("5.6.7.8"))
}

func TestRemoveLocation(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "example.com."
	locationStr := `{"a":{"ttl":300, "records":[{"ip":"1.2.3.4", "country":["ES"]},{"ip":"5.6.7.8", "country":[""]}]}}`
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	err = dh.SetLocationFromJson(zoneName, "www", locationStr)
	g.Expect(err).To(BeNil())
	_, err = dh.GetLocation(zoneName, "www")
	g.Expect(err).To(BeNil())
	err = dh.RemoveLocation(zoneName, "www")
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	location, err := dh.GetLocation(zoneName, "www")
	g.Expect(err).To(Equal(redisCon.ErrNil))
	g.Expect(location).To(BeNil())
}

func TestConfigUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	zoneName := "example.com."
	configStr := `{"cname_flattening":true, "domain_id":"12345"}`
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	err := dh.Clear()
	g.Expect(err).To(BeNil())
	err = dh.EnableZone(zoneName)
	g.Expect(err).To(BeNil())
	_, err = dh.GetZoneConfig(zoneName)
	g.Expect(err).To(BeNil())

	err = dh.SetZoneConfigFromJson(zoneName, configStr)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 1200)
	config, err := dh.GetZoneConfig(zoneName)
	g.Expect(err).To(BeNil())
	g.Expect(config.DomainId).To(Equal("12345"))
	g.Expect(config.CnameFlattening).To(BeTrue())
}

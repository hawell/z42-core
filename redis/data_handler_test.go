package redis

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/hawell/z42/types"
	"github.com/miekg/dns"
)

var dataHandlerDefaultTestConfig = DataHandlerConfig{
	ZoneCacheSize:      10000,
	ZoneCacheTimeout:   60,
	ZoneReload:         60,
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

func TestGetZone(t *testing.T) {
	dh := NewDataHandler(&dataHandlerDefaultTestConfig)
	_ = dh.Redis.Del("*")
	_ = dh.Redis.SAdd("z42:zones", "zone1.com")
	_ = dh.Redis.HSet("z42:zones:zone1.com.", "@", "{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"5.5.5.5\"}]}}")
	_ = dh.Redis.Set("z42:zones:zone1.com.:config", "{\"domain_id\":\"123456\", \"soa\":{\"ttl\":300, \"minttl\":100, \"mbox\":\"hostmaster.zone1.com.\",\"ns\":\"ns1.zone1.com.\",\"refresh\":44,\"retry\":55,\"expire\":66, \"serial\":32343}}")

	zone := dh.GetZone("zone1.com.")
	if zone == nil {
		fmt.Println("load zone failed")
		t.Fail()
		return
	}
	if zone.Name != "zone1.com." {
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
		DnsSec:          false,
		CnameFlattening: false,
	}
	if reflect.DeepEqual(zone.Config, zoneConfig) == false {
		fmt.Printf("%#v\n", zone.Config.SOA)
		fmt.Printf("%#v\n", zoneConfig.SOA)
		fmt.Println("config mismatch")
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
		Zone: nil,
		Name: "zone1.com.",
	}
	l := dh.GetLocation("zone1.com.", zone)
	if l.Name != location.Name {
		fmt.Println("location name mismatch")
		t.Fail()
	}
	if reflect.DeepEqual(l.A, location.A) == false {
		fmt.Println(l.A)
		fmt.Println(location.A)
		t.Fail()
	}
}

func TestLoadZones(t *testing.T) {
}

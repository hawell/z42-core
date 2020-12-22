package types

import (
	"bytes"
	iradix "github.com/hashicorp/go-immutable-radix"
	jsoniter "github.com/json-iterator/go"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"strings"
	"time"
)

type Zone struct {
	Name          string
	Config        *ZoneConfig
	LocationsTree *iradix.Tree
	LocationsList []string
	ZSK           *ZoneKey
	KSK           *ZoneKey
	DnsKeySig     dns.RR
	CacheTimeout  int64
}

type ZoneConfig struct {
	DomainId        string     `json:"domain_id,omitempty"`
	SOA             *SOA_RRSet `json:"soa,omitempty"`
	DnsSec          bool       `json:"dnssec,omitempty"`
	CnameFlattening bool       `json:"cname_flattening,omitempty"`
}

func ZoneConfigFromJson(zone string, configStr string) *ZoneConfig {
	config := &ZoneConfig{
		DnsSec:          false,
		CnameFlattening: false,
		SOA: &SOA_RRSet{
			Ns:      "ns1." + zone,
			MinTtl:  300,
			Refresh: 86400,
			Retry:   7200,
			Expire:  3600,
			MBox:    "hostmaster." + zone,
			Serial:  uint32(time.Now().Unix()),
			Ttl:     300,
		},
	}
	if len(configStr) > 0 {
		err := jsoniter.Unmarshal([]byte(configStr), config)
		if err != nil {
			zap.L().Error("cannot parse zone config", zap.Error(err))
		}
	}
	config.SOA.Ns = dns.Fqdn(config.SOA.Ns)
	config.SOA.Data = &dns.SOA{
		Hdr:     dns.RR_Header{Name: zone, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: config.SOA.Ttl, Rdlength: 0},
		Ns:      config.SOA.Ns,
		Mbox:    config.SOA.MBox,
		Refresh: config.SOA.Refresh,
		Retry:   config.SOA.Retry,
		Expire:  config.SOA.Expire,
		Minttl:  config.SOA.MinTtl,
		Serial:  config.SOA.Serial,
	}
	return config
}

func ReverseName(zone string) []byte {
	runes := []rune("." + zone)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return []byte(string(runes))
}

func NewZone(zone string, locations []string, configStr string) *Zone {
	z := new(Zone)
	z.Name = zone
	LocationsTree := iradix.New()
	rvalues := make([][]byte, 0, len(locations))
	for _, val := range locations {
		rvalues = append(rvalues, ReverseName(val))
	}
	for _, rvalue := range rvalues {
		for i := 0; i < len(rvalue); i++ {
			if rvalue[i] == '.' {
				if _, found := LocationsTree.Get(rvalue[:i+1]); !found {
					LocationsTree, _, _ = LocationsTree.Insert(rvalue[:i+1], nil)
				}
			}
		}
	}
	for i, rvalue := range rvalues {
		LocationsTree, _, _ = LocationsTree.Insert(rvalue, locations[i])
	}
	z.LocationsTree = LocationsTree
	z.LocationsList = locations

	z.Config = ZoneConfigFromJson(zone, configStr)

	return z
}

const (
	ExactMatch = iota
	WildCardMatch
	EmptyNonterminalMatch
	CEMatch
	NoMatch
)

func (z *Zone) FindLocation(query string) (string, int) {
	// request for zone records
	if query == z.Name {
		return "@", ExactMatch
	}

	query = strings.TrimSuffix(query, "."+z.Name)

	rquery := ReverseName(query)
	k, value, ok := z.LocationsTree.Root().LongestPrefix(rquery)
	prefix := make([]byte, len(k), len(k)+2)
	copy(prefix, k)
	if !ok {
		value, ok = z.LocationsTree.Get([]byte("*."))
		if ok && value != nil {
			return "*", WildCardMatch
		}
		return "", NoMatch
	}

	if value != nil {
		ce := value.(string)
		if bytes.Equal(prefix, rquery) {
			return query, ExactMatch
		} else {
			ss := append(prefix, []byte("*.")...)
			value, ok = z.LocationsTree.Get(ss)
			if ok && value != nil {
				return value.(string), WildCardMatch
			} else {
				return ce, CEMatch
			}
		}
	} else {
		if bytes.Equal(prefix, rquery) {
			return "", EmptyNonterminalMatch
		} else {
			ss := append(prefix, []byte("*.")...)
			value, ok = z.LocationsTree.Get(ss)
			if ok && value != nil {
				return value.(string), WildCardMatch
			} else {
				return "", NoMatch
			}
		}
	}
}

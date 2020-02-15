package handler

import (
	"bytes"
	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/hawell/logger"
	jsoniter "github.com/json-iterator/go"
	"github.com/miekg/dns"
	"strings"
	"time"
)

type Zone struct {
	Name         string
	Config       ZoneConfig
	Locations    *iradix.Tree
	ZSK          *ZoneKey
	KSK          *ZoneKey
	DnsKeySig    dns.RR
	CacheTimeout int64
}

type ZoneConfig struct {
	DomainId        string     `json:"domain_id,omitempty"`
	SOA             *SOA_RRSet `json:"soa,omitempty"`
	DnsSec          bool       `json:"dnssec,omitempty"`
	CnameFlattening bool       `json:"cname_flattening,omitempty"`
}

func reverseName(zone string) []byte {
	runes := []rune("." + zone)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return []byte(string(runes))
}

func NewZone(name string, locations []string, config string) *Zone {
	z := new(Zone)
	z.Name = name
	LocationsTree := iradix.New()
	rvalues := make([][]byte, 0, len(locations))
	for _, val := range locations {
		rvalues = append(rvalues, reverseName(val))
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
	z.Locations = LocationsTree

	z.Config = ZoneConfig{
		DnsSec:          false,
		CnameFlattening: false,
		SOA: &SOA_RRSet{
			Ns:      "ns1." + z.Name,
			MinTtl:  300,
			Refresh: 86400,
			Retry:   7200,
			Expire:  3600,
			MBox:    "hostmaster." + z.Name,
			Serial:  uint32(time.Now().Unix()),
			Ttl:     300,
		},
	}
	if len(config) > 0 {
		err := jsoniter.Unmarshal([]byte(config), &z.Config)
		if err != nil {
			logger.Default.Errorf("cannot parse zone config : %s", err)
		}
	}
	z.Config.SOA.Ns = dns.Fqdn(z.Config.SOA.Ns)
	z.Config.SOA.Data = &dns.SOA{
		Hdr:     dns.RR_Header{Name: z.Name, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: z.Config.SOA.Ttl, Rdlength: 0},
		Ns:      z.Config.SOA.Ns,
		Mbox:    z.Config.SOA.MBox,
		Refresh: z.Config.SOA.Refresh,
		Retry:   z.Config.SOA.Retry,
		Expire:  z.Config.SOA.Expire,
		Minttl:  z.Config.SOA.MinTtl,
		Serial:  z.Config.SOA.Serial,
	}
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
		return query, ExactMatch
	}

	query = strings.TrimSuffix(query, "."+z.Name)

	rquery := reverseName(query)
	k, value, ok := z.Locations.Root().LongestPrefix(rquery)
	prefix := make([]byte, len(k), len(k)+2)
	copy(prefix, k)
	if !ok {
		value, ok = z.Locations.Get([]byte("*."))
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
			value, ok = z.Locations.Get(ss)
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
			value, ok = z.Locations.Get(ss)
			if ok && value != nil {
				return value.(string), WildCardMatch
			} else {
				return "", NoMatch
			}
		}
	}
}

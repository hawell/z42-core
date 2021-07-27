package types

import (
	"crypto"
	"github.com/miekg/dns"
	"net"
)

const (
	IpMaskWhite = iota
	IpMaskGrey
	IpMaskBlack
)

const (
	TypeANAME = 500
)

var SupportedTypes = map[string]struct{}{"a": {}, "aaaa": {}, "cname": {}, "txt": {}, "ns": {}, "mx": {}, "srv": {}, "caa": {}, "ptr": {}, "tlsa": {}, "ds": {}, "aname": {}, "soa": {}}

var TypeToRRSet = map[string]func() RRSet {
	"a": func() RRSet { return new(IP_RRSet) },
	"aaaa": func() RRSet { return new(IP_RRSet) },
	"cname": func() RRSet { return new(CNAME_RRSet) },
	"txt": func() RRSet { return new(TXT_RRSet) },
	"ns": func() RRSet { return new(NS_RRSet) },
	"mx": func() RRSet { return new(MX_RRSet) },
	"srv": func() RRSet { return new(SRV_RRSet) },
	"caa": func() RRSet { return new(CAA_RRSet) },
	"ptr": func() RRSet { return new(PTR_RRSet) },
	"tlsa": func() RRSet { return new(TLSA_RRSet) },
	"ds": func() RRSet { return new(DS_RRSet) },
	"soa": func() RRSet { return new(SOA_RRSet) },
	"aname": func() RRSet { return new(ANAME_RRSet) },
}

type RRSet interface {
	Value(name string) []dns.RR
	Empty() bool
	Ttl() uint32
}

type GenericRRSet struct {
	TtlValue uint32 `json:"ttl,omitempty"`
}

func (rrset *GenericRRSet) Ttl() uint32 {
	return rrset.TtlValue
}

type ZoneKey struct {
	DnsKey        *dns.DNSKEY
	PrivateKey    crypto.PrivateKey
	KeyInception  uint32
	KeyExpiration uint32
}

type IP_RR struct {
	Weight  int      `json:"weight,omitempty"`
	Ip      net.IP   `json:"ip"`
	Country []string `json:"country,omitempty"`
	ASN     []uint   `json:"asn,omitempty"`
}

type IpHealthCheckConfig struct {
	Protocol  string `json:"protocol,omitempty"`
	Uri       string `json:"uri,omitempty"`
	Port      int    `json:"port,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
	UpCount   int    `json:"up_count,omitempty"`
	DownCount int    `json:"down_count,omitempty"`
	Enable    bool   `json:"enable,omitempty"`
}

type IpFilterConfig struct {
	Count     string `json:"count,omitempty"`      // "multi", "single"
	Order     string `json:"order,omitempty"`      // "weighted", "rr", "none"
	GeoFilter string `json:"geo_filter,omitempty"` // "country", "location", "asn", "asn+country", "none"
}

type IP_RRSet struct {
	GenericRRSet
	FilterConfig      IpFilterConfig      `json:"filter,omitempty"`
	HealthCheckConfig IpHealthCheckConfig `json:"health_check,omitempty"`
	Data              []IP_RR             `json:"records,omitempty"`
}

func (*IP_RRSet) Value(string) []dns.RR {
	return nil
}

func (rrset *IP_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type CNAME_RRSet struct {
	GenericRRSet
	Host string `json:"host"`
}

func (rrset *CNAME_RRSet) Value(name string) []dns.RR {
	if len(rrset.Host) == 0 {
		return []dns.RR{}
	}
	r := new(dns.CNAME)
	r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME,
		Class: dns.ClassINET, Ttl: rrset.TtlValue}
	r.Target = dns.Fqdn(rrset.Host)
	return []dns.RR{r}
}

func (rrset *CNAME_RRSet) Empty() bool {
	return len(rrset.Host) == 0
}

type TXT_RR struct {
	Text string `json:"text"`
}

type TXT_RRSet struct {
	GenericRRSet
	Data []TXT_RR `json:"records,omitempty"`
}

func (rrset *TXT_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, txt := range rrset.Data {
		if len(txt.Text) == 0 {
			continue
		}
		r := new(dns.TXT)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeTXT,
			Class: dns.ClassINET, Ttl: rrset.TtlValue}
		r.Txt = split255(txt.Text)
		res = append(res, r)
	}
	return res
}

func (rrset *TXT_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type NS_RR struct {
	Host string `json:"host"`
}

type NS_RRSet struct {
	GenericRRSet
	Data []NS_RR `json:"records,omitempty"`
}

func (rrset *NS_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, ns := range rrset.Data {
		if len(ns.Host) == 0 {
			continue
		}
		r := new(dns.NS)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeNS,
			Class: dns.ClassINET, Ttl: rrset.TtlValue}
		r.Ns = dns.Fqdn(ns.Host)
		res = append(res, r)
	}
	return res
}

func (rrset *NS_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type MX_RR struct {
	Host       string `json:"host"`
	Preference uint16 `json:"preference"`
}

type MX_RRSet struct {
	GenericRRSet
	Data []MX_RR `json:"records,omitempty"`
}

func (rrset *MX_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, mx := range rrset.Data {
		if len(mx.Host) == 0 {
			continue
		}
		r := new(dns.MX)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeMX,
			Class: dns.ClassINET, Ttl: rrset.TtlValue}
		r.Mx = dns.Fqdn(mx.Host)
		r.Preference = mx.Preference
		res = append(res, r)
	}
	return res
}

func (rrset *MX_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type SRV_RR struct {
	Target   string `json:"target"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
	Port     uint16 `json:"port"`
}

type SRV_RRSet struct {
	GenericRRSet
	Data []SRV_RR `json:"records,omitempty"`
}

func (rrset *SRV_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, srv := range rrset.Data {
		if len(srv.Target) == 0 {
			continue
		}
		r := new(dns.SRV)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeSRV,
			Class: dns.ClassINET, Ttl: rrset.TtlValue}
		r.Target = dns.Fqdn(srv.Target)
		r.Weight = srv.Weight
		r.Port = srv.Port
		r.Priority = srv.Priority
		res = append(res, r)
	}
	return res
}

func (rrset *SRV_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type CAA_RRSet struct {
	GenericRRSet
	Data []CAA_RR `json:"records,omitempty"`
}

type CAA_RR struct {
	Tag   string `json:"tag"`
	Value string `json:"value"`
	Flag  uint8  `json:"flag"`
}

func (rrset *CAA_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, caa := range rrset.Data {
		r := new(dns.CAA)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCAA,
			Class: dns.ClassINET, Ttl: rrset.TtlValue}
		r.Value = caa.Value
		r.Flag = caa.Flag
		r.Tag = caa.Tag
		res = append(res, r)
	}
	return res
}

func (rrset *CAA_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type PTR_RRSet struct {
	GenericRRSet
	Domain string `json:"domain"`
}

func (rrset *PTR_RRSet) Value(name string) []dns.RR {
	if len(rrset.Domain) == 0 {
		return []dns.RR{}
	}
	r := new(dns.PTR)
	r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypePTR,
		Class: dns.ClassINET, Ttl: rrset.TtlValue}
	r.Ptr = dns.Fqdn(rrset.Domain)
	return []dns.RR{r}
}

func (rrset *PTR_RRSet) Empty() bool {
	return len(rrset.Domain) == 0
}

type TLSA_RR struct {
	Usage        uint8  `json:"usage"`
	Selector     uint8  `json:"selector"`
	MatchingType uint8  `json:"matching_type"`
	Certificate  string `json:"certificate"`
}

type TLSA_RRSet struct {
	GenericRRSet
	Data []TLSA_RR `json:"records,omitempty"`
}

func (rrset *TLSA_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, tlsa := range rrset.Data {
		r := new(dns.TLSA)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeTLSA,
			Class: dns.ClassNONE, Ttl: rrset.TtlValue}
		r.Usage = tlsa.Usage
		r.Selector = tlsa.Selector
		r.MatchingType = tlsa.MatchingType
		r.Certificate = tlsa.Certificate
		res = append(res, r)
	}
	return res
}

func (rrset *TLSA_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type DS_RR struct {
	KeyTag     uint16 `json:"key_tag"`
	Algorithm  uint8  `json:"algorithm"`
	DigestType uint8  `json:"digest_type"`
	Digest     string `json:"digest"`
}

type DS_RRSet struct {
	GenericRRSet
	Data []DS_RR `json:"records,omitempty"`
}

func (rrset *DS_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, ds := range rrset.Data {
		r := new(dns.DS)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeDS,
			Class: dns.ClassINET, Ttl: rrset.TtlValue}
		r.KeyTag = ds.KeyTag
		r.Algorithm = ds.Algorithm
		r.DigestType = ds.DigestType
		r.Digest = ds.Digest
		res = append(res, r)
	}
	return res
}

func (rrset *DS_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

type SOA_RRSet struct {
	GenericRRSet
	Ns      string   `json:"ns"`
	MBox    string   `json:"mbox"`
	Data    *dns.SOA `json:"-"`
	Refresh uint32   `json:"refresh"`
	Retry   uint32   `json:"retry"`
	Expire  uint32   `json:"expire"`
	MinTtl  uint32   `json:"minttl"`
	Serial  uint32   `json:"serial"`
}

func (rrset *SOA_RRSet) Value(name string) []dns.RR {
	r := new(dns.SOA)
	r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeSOA,
		Class: dns.ClassINET, Ttl: rrset.TtlValue}
	r.Ns = rrset.Ns
	r.Mbox = rrset.MBox
	r.Refresh = rrset.Refresh
	r.Retry = rrset.Retry
	r.Expire = rrset.Expire
	r.Minttl = rrset.MinTtl
	r.Serial = rrset.Serial
	return nil
}

func (rrset *SOA_RRSet) Empty() bool {
	return false
}

type ANAME_RRSet struct {
	GenericRRSet
	Location string `json:"location,omitempty"`
}

func (*ANAME_RRSet) Value(string) []dns.RR {
	return nil
}

func (rrset *ANAME_RRSet) Empty() bool {
	return len(rrset.Location) == 0
}

type RRSetKey struct {
	QName string
	QType uint16
}

func SplitSets(rrs []dns.RR) map[RRSetKey][]dns.RR {
	m := make(map[RRSetKey][]dns.RR)

	for _, r := range rrs {
		if r.Header().Rrtype == dns.TypeRRSIG || r.Header().Rrtype == dns.TypeOPT {
			continue
		}

		if s, ok := m[RRSetKey{r.Header().Name, r.Header().Rrtype}]; ok {
			s = append(s, r)
			m[RRSetKey{r.Header().Name, r.Header().Rrtype}] = s
			continue
		}

		s := make([]dns.RR, 1, 3)
		s[0] = r
		m[RRSetKey{r.Header().Name, r.Header().Rrtype}] = s
	}

	if len(m) > 0 {
		return m
	}
	return nil
}

func split255(s string) []string {
	if len(s) < 255 {
		return []string{s}
	}
	var sx []string
	p, i := 0, 255
	for {
		if i <= len(s) {
			sx = append(sx, s[p:i])
		} else {
			sx = append(sx, s[p:])
			break

		}
		p, i = p+255, i+255
	}

	return sx
}

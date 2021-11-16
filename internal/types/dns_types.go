package types

import (
	"crypto"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	IpMaskWhite = iota
	IpMaskGrey
	IpMaskBlack
)

const (
	TypeANAME = 500
)

func IsSupported(t uint16) bool {
	switch t {
	case dns.TypeA,
		dns.TypeAAAA,
		dns.TypeCNAME,
		dns.TypeTXT,
		dns.TypeNS,
		dns.TypeMX,
		dns.TypeSRV,
		dns.TypeCAA,
		dns.TypePTR,
		dns.TypeTLSA,
		dns.TypeDS,
		TypeANAME,
		dns.TypeSOA:
		return true
	default:
		return false
	}
}

func TypeToRRSet(t uint16) RRSet {
	switch t {
	case dns.TypeA:
		return &IP_RRSet{Data: []IP_RR{}}
	case dns.TypeAAAA:
		return &IP_RRSet{Data: []IP_RR{}}
	case dns.TypeCNAME:
		return &CNAME_RRSet{}
	case dns.TypeTXT:
		return &TXT_RRSet{Data: []TXT_RR{}}
	case dns.TypeNS:
		return &NS_RRSet{Data: []NS_RR{}}
	case dns.TypeMX:
		return &MX_RRSet{Data: []MX_RR{}}
	case dns.TypeSRV:
		return &SRV_RRSet{Data: []SRV_RR{}}
	case dns.TypeCAA:
		return &CAA_RRSet{Data: []CAA_RR{}}
	case dns.TypePTR:
		return &PTR_RRSet{}
	case dns.TypeTLSA:
		return &TLSA_RRSet{Data: []TLSA_RR{}}
	case dns.TypeDS:
		return &DS_RRSet{Data: []DS_RR{}}
	case dns.TypeSOA:
		return &SOA_RRSet{}
	case TypeANAME:
		return &ANAME_RRSet{}
	default:
		return nil
	}
}

func TypeStrToRRSet(t string) RRSet {
	return TypeToRRSet(StringToType(t))
}

func StringToType(s string) uint16 {
	s = strings.ToUpper(s)
	if s == "ANAME" {
		return TypeANAME
	}
	return dns.StringToType[s]
}

var customTypeToString = map[uint16]string{
	TypeANAME: "ANAME",
}

func TypeToString(t uint16) string {
	s, ok := customTypeToString[t]
	if !ok {
		s = dns.TypeToString[t]
	}
	return strings.ToLower(s)
}

type RRSet interface {
	Value(name string) []dns.RR
	Empty() bool
	Ttl() uint32
	Parse(rr dns.RR) error
}

var errInvalidType = errors.New("invalid type")

func String(rrset RRSet, name string) string {
	v := rrset.Value("")
	if len(v) == 0 {
		return ""
	}
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s%s\n", name, v[0].String())
	for i := 1; i < len(v); i++ {
		_, _ = fmt.Fprintf(&b, "\t%s\n", v[i].String())
	}
	return b.String()
}

type GenericRRSet struct {
	TtlValue uint32 `json:"ttl,default:300"`
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
	Data              []IP_RR             `json:"records"`
}

func (rrset *IP_RRSet) Value(name string) []dns.RR {
	var res []dns.RR
	for _, record := range rrset.Data {
		if record.Ip.String() == "" {
			continue
		}
		if record.Ip.To4() != nil {
			r := new(dns.A)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA,
				Class: dns.ClassINET, Ttl: rrset.TtlValue}
			r.A = record.Ip
			res = append(res, r)
		} else {
			r := new(dns.AAAA)
			r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA,
				Class: dns.ClassINET, Ttl: rrset.TtlValue}
			r.AAAA = record.Ip
			res = append(res, r)
		}
	}
	return res
}

func (rrset *IP_RRSet) Empty() bool {
	return len(rrset.Data) == 0
}

func (rrset *IP_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeA && r.Header().Rrtype != dns.TypeAAAA {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	if r.Header().Rrtype == dns.TypeA {
		rrset.Data = append(rrset.Data, IP_RR{Ip: r.(*dns.A).A})
	} else {
		rrset.Data = append(rrset.Data, IP_RR{Ip: r.(*dns.AAAA).AAAA})
	}
	return nil
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

func (rrset *CNAME_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeCNAME {
		return errInvalidType
	}
	rrset.TtlValue = r.Header().Ttl
	rrset.Host = r.(*dns.CNAME).Target
	return nil
}

type TXT_RR struct {
	Text string `json:"text"`
}

type TXT_RRSet struct {
	GenericRRSet
	Data []TXT_RR `json:"records"`
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

func (rrset *TXT_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeTXT {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	rrset.Data = append(rrset.Data, TXT_RR{Text: strings.Join(r.(*dns.TXT).Txt, "")})
	return nil
}

type NS_RR struct {
	Host string `json:"host"`
}

var nsNames = []string{"alpha", "bravo", "charlie", "echo", "foxtrot", "golf", "hotel", "india", "Juliet", "kilo", "lima", "mike", "november", "oscar", "papa", "quebec", "romeo", "sierra", "tango", "uniform", "victor", "whiskey", "xray", "yankee", "zulu"}

func GenerateNS(authServer string) *NS_RRSet {
	return &NS_RRSet{
		GenericRRSet: GenericRRSet{TtlValue: 3600},
		Data: []NS_RR{
			{Host: nsNames[rand.Int()%len(nsNames)] + "." + authServer},
			{Host: nsNames[rand.Int()%len(nsNames)] + "." + authServer},
		},
	}
}

type NS_RRSet struct {
	GenericRRSet
	Data []NS_RR `json:"records"`
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

func (rrset *NS_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeNS {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	rrset.Data = append(rrset.Data, NS_RR{Host: r.(*dns.NS).Ns})
	return nil
}

type MX_RR struct {
	Host       string `json:"host"`
	Preference uint16 `json:"preference"`
}

type MX_RRSet struct {
	GenericRRSet
	Data []MX_RR `json:"records"`
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

func (rrset *MX_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeMX {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	rrset.Data = append(rrset.Data, MX_RR{Host: r.(*dns.MX).Mx, Preference: r.(*dns.MX).Preference})
	return nil
}

type SRV_RR struct {
	Target   string `json:"target"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
	Port     uint16 `json:"port"`
}

type SRV_RRSet struct {
	GenericRRSet
	Data []SRV_RR `json:"records"`
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

func (rrset *SRV_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeSRV {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	srv := r.(*dns.SRV)
	rrset.Data = append(rrset.Data, SRV_RR{
		Target:   srv.Target,
		Priority: srv.Priority,
		Weight:   srv.Weight,
		Port:     srv.Port,
	})
	return nil
}

type CAA_RRSet struct {
	GenericRRSet
	Data []CAA_RR `json:"records"`
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

func (rrset *CAA_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeCAA {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	caa := r.(*dns.CAA)
	rrset.Data = append(rrset.Data, CAA_RR{
		Tag:   caa.Tag,
		Value: caa.Value,
		Flag:  caa.Flag,
	})
	return nil
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

func (rrset *PTR_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypePTR {
		return errInvalidType
	}
	rrset.TtlValue = r.Header().Ttl
	rrset.Domain = r.(*dns.PTR).Ptr
	return nil
}

type TLSA_RR struct {
	Usage        uint8  `json:"usage"`
	Selector     uint8  `json:"selector"`
	MatchingType uint8  `json:"matching_type"`
	Certificate  string `json:"certificate"`
}

type TLSA_RRSet struct {
	GenericRRSet
	Data []TLSA_RR `json:"records"`
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

func (rrset *TLSA_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeTLSA {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	tlsa := r.(*dns.TLSA)
	rrset.Data = append(rrset.Data, TLSA_RR{
		Usage:        tlsa.Usage,
		Selector:     tlsa.Selector,
		MatchingType: tlsa.MatchingType,
		Certificate:  tlsa.Certificate,
	})
	return nil
}

type DS_RR struct {
	KeyTag     uint16 `json:"key_tag"`
	Algorithm  uint8  `json:"algorithm"`
	DigestType uint8  `json:"digest_type"`
	Digest     string `json:"digest"`
}

type DS_RRSet struct {
	GenericRRSet
	Data []DS_RR `json:"records"`
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

func (rrset *DS_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeDS {
		return errInvalidType
	}
	if len(rrset.Data) == 0 {
		rrset.TtlValue = r.Header().Ttl
	}
	ds := r.(*dns.DS)
	rrset.Data = append(rrset.Data, DS_RR{
		KeyTag:     ds.KeyTag,
		Algorithm:  ds.Algorithm,
		DigestType: ds.DigestType,
		Digest:     ds.Digest,
	})
	return nil
}

func DefaultSOA(zoneName string) *SOA_RRSet {
	serialStr := time.Now().Format("20060102") + "00"
	serial, _ := strconv.Atoi(serialStr)
	return &SOA_RRSet{
		GenericRRSet: GenericRRSet{TtlValue: 3600},
		Ns:           "ns1." + zoneName,
		MBox:         "hostmaster." + zoneName,
		Refresh:      86400,
		Retry:        7200,
		Expire:       3600000,
		MinTtl:       300,
		Serial:       uint32(serial),
	}
}

type SOA_RRSet struct {
	GenericRRSet
	Ns      string   `json:"ns"`
	MBox    string   `json:"mbox"`
	Data    *dns.SOA `json:"-"`
	Refresh uint32   `json:"refresh,default:86400"`
	Retry   uint32   `json:"retry,default:7200"`
	Expire  uint32   `json:"expire,default:3600000"`
	MinTtl  uint32   `json:"minttl,default:300"`
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
	return []dns.RR{r}
}

func (rrset *SOA_RRSet) Empty() bool {
	return false
}

func (rrset *SOA_RRSet) Parse(r dns.RR) error {
	if r.Header().Rrtype != dns.TypeSOA {
		return errInvalidType
	}
	rrset.TtlValue = r.Header().Ttl
	soa := r.(*dns.SOA)
	rrset.Ns = soa.Ns
	rrset.Expire = soa.Expire
	rrset.MBox = soa.Mbox
	rrset.MinTtl = soa.Minttl
	rrset.Refresh = soa.Refresh
	rrset.Retry = soa.Retry
	rrset.Serial = soa.Serial
	return nil
}

type ANAME_RRSet struct {
	GenericRRSet
	Location string `json:"location"`
}

func (rrset *ANAME_RRSet) Value(name string) []dns.RR {
	r := new(dns.CNAME)
	r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME,
		Class: dns.ClassINET, Ttl: rrset.TtlValue}
	r.Target = rrset.Location
	return []dns.RR{r}
}

func (rrset *ANAME_RRSet) Empty() bool {
	return len(rrset.Location) == 0
}

func (rrset *ANAME_RRSet) Parse(dns.RR) error {
	return errInvalidType
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

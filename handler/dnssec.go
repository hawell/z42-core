package handler

import (
	"crypto/ecdsa"
	"crypto/rsa"

	"github.com/hawell/logger"
	"github.com/miekg/dns"
)

var (
	NsecBitmapZone          = []uint16{dns.TypeA, dns.TypeCNAME, dns.TypePTR, dns.TypeMX, dns.TypeTXT, dns.TypeAAAA, dns.TypeSRV, dns.TypeRRSIG, dns.TypeNSEC, dns.TypeTLSA, dns.TypeCAA}
	NsecBitmapAppex         = []uint16{dns.TypeA, dns.TypeNS, dns.TypeSOA, dns.TypePTR, dns.TypeMX, dns.TypeTXT, dns.TypeAAAA, dns.TypeSRV, dns.TypeRRSIG, dns.TypeNSEC, dns.TypeTLSA, dns.TypeCAA}
	NsecBitmapSubDelegation = []uint16{dns.TypeNS, dns.TypeDS, dns.TypeRRSIG, dns.TypeNSEC}
	NsecBitmapNameError     = []uint16{dns.TypeRRSIG, dns.TypeNSEC}
)

func FilterNsecBitmap(qtype uint16, bitmap []uint16) []uint16 {
	res := make([]uint16, 0, len(bitmap))
	for i := range bitmap {
		if bitmap[i] != qtype {
			res = append(res, bitmap[i])
		}
	}
	return res
}

type rrset struct {
	qname string
	qtype uint16
}

func splitSets(rrs []dns.RR) map[rrset][]dns.RR {
	m := make(map[rrset][]dns.RR)

	for _, r := range rrs {
		if r.Header().Rrtype == dns.TypeRRSIG || r.Header().Rrtype == dns.TypeOPT {
			continue
		}

		if s, ok := m[rrset{r.Header().Name, r.Header().Rrtype}]; ok {
			s = append(s, r)
			m[rrset{r.Header().Name, r.Header().Rrtype}] = s
			continue
		}

		s := make([]dns.RR, 1, 3)
		s[0] = r
		m[rrset{r.Header().Name, r.Header().Rrtype}] = s
	}

	if len(m) > 0 {
		return m
	}
	return nil
}

func Sign(rrs []dns.RR, qname string, z *Zone) []dns.RR {
	var res []dns.RR
	sets := splitSets(rrs)
	for _, set := range sets {
		res = append(res, set...)
		switch set[0].Header().Rrtype {
		case dns.TypeRRSIG, dns.TypeOPT:
			continue
		case dns.TypeDNSKEY:
			res = append(res, z.DnsKeySig)
		case dns.TypeNS:
			if qname == z.Name {
				res = append(res, sign(set, set[0].Header().Name, z.ZSK, set[0].Header().Ttl))
			}
		default:
			res = append(res, sign(set, set[0].Header().Name, z.ZSK, set[0].Header().Ttl))
		}
	}
	return res
}

func sign(rrs []dns.RR, name string, key *ZoneKey, ttl uint32) *dns.RRSIG {
	rrsig := &dns.RRSIG{
		Hdr:        dns.RR_Header{Name: name, Rrtype: dns.TypeRRSIG, Class: dns.ClassINET, Ttl: ttl},
		Inception:  key.KeyInception,
		Expiration: key.KeyExpiration,
		KeyTag:     key.DnsKey.KeyTag(),
		SignerName: key.DnsKey.Hdr.Name,
		Algorithm:  key.DnsKey.Algorithm,
	}
	switch rrsig.Algorithm {
	case dns.RSAMD5, dns.RSASHA1, dns.RSASHA1NSEC3SHA1, dns.RSASHA256, dns.RSASHA512:
		if err := rrsig.Sign(key.PrivateKey.(*rsa.PrivateKey), rrs); err != nil {
			logger.Default.Errorf("sign failed : %s", err)
			return nil
		}
	case dns.ECDSAP256SHA256, dns.ECDSAP384SHA384:
		if err := rrsig.Sign(key.PrivateKey.(*ecdsa.PrivateKey), rrs); err != nil {
			logger.Default.Errorf("sign failed : %s", err)
			return nil
		}
	case dns.DSA, dns.DSANSEC3SHA1:
		//rrsig.Sign(zone.PrivateKey.(*dsa.PrivateKey), rrs)
		fallthrough
	default:
		return nil
	}
	return rrsig
}

func NSec(context *RequestContext, name string, qtype uint16, zone *Zone) {
	if !context.dnssec {
		return
	}
	var bitmap []uint16
	if name == zone.Name {
		context.Res = dns.RcodeSuccess
		bitmap = FilterNsecBitmap(qtype, NsecBitmapAppex)
	} else {
		if context.Res == dns.RcodeNameError {
			context.Res = dns.RcodeSuccess
			bitmap = NsecBitmapNameError
		} else {
			if qtype == dns.TypeDS {
				bitmap = FilterNsecBitmap(qtype, NsecBitmapSubDelegation)
			} else {
				bitmap = FilterNsecBitmap(qtype, NsecBitmapZone)
			}
		}
	}

	nsec := &dns.NSEC{
		Hdr:        dns.RR_Header{Name: name, Rrtype: dns.TypeNSEC, Class: dns.ClassINET, Ttl: zone.Config.SOA.MinTtl},
		NextDomain: "\\000." + name,
		TypeBitMap: bitmap,
	}
	context.Authority = append(context.Authority, nsec)
}

func ApplyDnssec(context *RequestContext, zone *Zone) {
	if !context.dnssec {
		return
	}
	context.Answer = Sign(context.Answer, context.RawName(), zone)
	context.Authority = Sign(context.Authority, context.RawName(), zone)
	// context.Additional = Sign(context.Additional, context.RawName(), zone)

}

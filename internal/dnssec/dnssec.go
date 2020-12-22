package dnssec

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"github.com/hawell/z42/internal/types"
	"go.uber.org/zap"

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

func SignResponse(rrs []dns.RR, qname string, z *types.Zone) []dns.RR {
	var res []dns.RR
	sets := types.SplitSets(rrs)
	for _, set := range sets {
		res = append(res, set...)
		switch set[0].Header().Rrtype {
		case dns.TypeRRSIG, dns.TypeOPT:
			continue
		case dns.TypeDNSKEY:
			res = append(res, z.DnsKeySig)
		case dns.TypeNS:
			if qname == z.Name {
				res = append(res, SignRRSet(set, set[0].Header().Name, z.ZSK, set[0].Header().Ttl))
			}
		default:
			res = append(res, SignRRSet(set, set[0].Header().Name, z.ZSK, set[0].Header().Ttl))
		}
	}
	return res
}

func SignRRSet(rrs []dns.RR, name string, key *types.ZoneKey, ttl uint32) *dns.RRSIG {
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
			zap.L().Error("sign failed", zap.Error(err))
			return nil
		}
	case dns.ECDSAP256SHA256, dns.ECDSAP384SHA384:
		if err := rrsig.Sign(key.PrivateKey.(*ecdsa.PrivateKey), rrs); err != nil {
			zap.L().Error("sign failed", zap.Error(err))
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

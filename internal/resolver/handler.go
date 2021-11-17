package resolver

import (
	"github.com/hawell/z42/internal/geotools"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/geoip"
	"go.uber.org/zap"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hawell/z42/internal/dnssec"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/internal/upstream"

	"github.com/miekg/dns"
)

type DnsRequestHandler struct {
	Config        *Config
	RedisData     *storage.DataHandler
	requestLogger *zap.Logger
	geoip         *geoip.GeoIp
	upstream      *upstream.Upstream
	quit          chan struct{}
	quitWG        sync.WaitGroup
}

func NewHandler(config *Config, redisData *storage.DataHandler, requestLogger *zap.Logger) *DnsRequestHandler {
	h := &DnsRequestHandler{
		Config:        config,
		RedisData:     redisData,
		requestLogger: requestLogger,
	}

	h.geoip = geoip.NewGeoIp(&config.GeoIp)
	h.upstream = upstream.NewUpstream(config.Upstream)
	h.quit = make(chan struct{})

	return h
}

func (h *DnsRequestHandler) ShutDown() {
	zap.L().Debug("handler : stopping")
	close(h.quit)
	h.quitWG.Wait()
	zap.L().Debug("handler : stopped")
}

func (h *DnsRequestHandler) response(context *RequestContext) {
	h.logRequest(context)
	context.Response()
}

func (h *DnsRequestHandler) HandleRequest(context *RequestContext) {
	zap.L().Debug(
		"start handle request",
		zap.Uint16("id", context.Req.Id),
		zap.String("query", context.RawName()),
		zap.String("type", context.Type()),
	)
	if h.Config.LogSourceLocation {
		sourceIP := context.SourceIp
		context.SourceCountry, _ = h.geoip.GetCountry(sourceIP)
		context.SourceASN, _ = h.geoip.GetASN(sourceIP)
	}

	zoneName := h.RedisData.FindZone(context.RawName())
	if zoneName == "" {
		zap.L().Debug(
			"zone not found",
			zap.Uint16("id", context.Req.Id),
		)
		context.Res = dns.RcodeNotAuth
		h.response(context)
		return
	}
	zap.L().Debug(
		"zone matched",
		zap.Uint16("id", context.Req.Id),
		zap.String("zone", zoneName),
	)

	context.zone = h.RedisData.GetZone(zoneName)
	if context.zone == nil {
		context.Res = dns.RcodeServerFailure
		h.response(context)
		return
	}
	context.DomainUid = context.zone.Config.DomainId

	context.dnssec = context.Do() && context.Auth && context.zone.Config.DnsSec
	cnameFlattening := context.dnssec || context.zone.Config.CnameFlattening

	loopCount := 0
	currentQName := context.RawName()
loop:
	for {
		if loopCount > 10 {
			zap.L().Error(
				"CNAME loop in request",
				zap.String("query", context.RawName()),
				zap.String("type", context.Type()),
			)
			context.Answer = []dns.RR{}
			context.Res = dns.RcodeServerFailure
			break loop
		}
		loopCount++

		if h.RedisData.FindZone(currentQName) != zoneName {
			zap.L().Debug(
				"out of zone",
				zap.Uint16("id", context.Req.Id),
				zap.String("qname", currentQName),
			)
			context.Res = dns.RcodeSuccess
			if len(context.Answer) == 0 {
				addNSec(context, context.RawName(), context.QType())
			}
			break loop
		}

		location, match := context.zone.FindLocation(currentQName)
		switch match {
		case types.NoMatch:
			zap.L().Debug(
				"no location matched",
				zap.Uint16("id", context.Req.Id),
				zap.String("qname", currentQName),
			)
			context.Authority = []dns.RR{context.zone.Config.SOA.Data}
			context.Res = dns.RcodeNameError
			addNSec(context, currentQName, dns.TypeNone)
			break loop

		case types.EmptyNonterminalMatch:
			zap.L().Debug(
				"empty non-terminal matched",
				zap.Uint16("id", context.Req.Id),
				zap.String("qname", currentQName),
				zap.String("location", location),
			)
			context.Authority = []dns.RR{context.zone.Config.SOA.Data}
			context.Res = dns.RcodeSuccess
			addNSec(context, currentQName, dns.TypeNone)
			break loop

		case types.CEMatch:
			zap.L().Debug(
				"ce math",
				zap.Uint16("id", context.Req.Id),
				zap.String("qname", currentQName),
				zap.String("location", location),
			)
			ns, err := h.RedisData.NS(context.zone.Name, location)
			if err != nil {
				context.Res = dns.RcodeServerFailure
				break loop
			}
			if !ns.Empty() {
				zap.L().Debug(
					"sub delegation",
					zap.Uint16("id", context.Req.Id),
					zap.String("qname", currentQName),
					zap.String("location", location),
				)
				if len(ns.Data) == 0 {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				cutPoint := location + "." + zoneName
				context.Authority = append(context.Authority, ns.Value(cutPoint)...)
				ds, err := h.RedisData.DS(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				if ds.Empty() {
					addNSec(context, cutPoint, dns.TypeDS)
				}
				context.Authority = append(context.Authority, ds.Value(cutPoint)...)
				for _, ns := range ns.Data {
					glueLocation, match := context.zone.FindLocation(ns.Host)
					if match != types.NoMatch {
						glueA, err := h.RedisData.A(context.zone.Name, glueLocation)
						// XXX : should we return with RcodeServerFailure?
						if err == nil {
							ips := h.filter(context.SourceIp, glueA)
							context.Additional = append(context.Additional, generateA(ns.Host, glueA.Ttl(), ips)...)
						}
						glueAAAA, err := h.RedisData.AAAA(context.zone.Name, glueLocation)
						if err == nil {
							ips := h.filter(context.SourceIp, glueAAAA)
							context.Additional = append(context.Additional, generateAAAA(ns.Host, glueAAAA.Ttl(), ips)...)
						}
					}
				}
				context.Res = dns.RcodeSuccess
				break loop
			} else {
				context.Authority = []dns.RR{context.zone.Config.SOA.Data}
				context.Res = dns.RcodeNameError
				addNSec(context, currentQName, context.QType())
				break loop
			}

		case types.WildCardMatch:
			fallthrough

		case types.ExactMatch:
			zap.L().Debug(
				"match",
				zap.Uint16("id", context.Req.Id),
				zap.String("qname", currentQName),
				zap.String("location", location),
			)
			cname, err := h.RedisData.CNAME(context.zone.Name, location)
			if err != nil {
				context.Res = dns.RcodeServerFailure
				break loop
			}
			if !cname.Empty() && context.QType() != dns.TypeCNAME {
				zap.L().Debug(
					"cname chain",
					zap.Uint16("id", context.Req.Id),
					zap.String("source", currentQName),
					zap.String("destination", cname.Host),
				)
				if !cnameFlattening {
					context.Answer = append(context.Answer, cname.Value(currentQName)...)
				} else if h.RedisData.FindZone(cname.Host) != zoneName {
					context.Answer = append(context.Answer, cname.Value(context.RawName())...)
					context.Res = dns.RcodeSuccess
					break loop
				}
				currentQName = dns.Fqdn(cname.Host)
				continue
			}
			if currentQName != context.zone.Name {
				ns, err := h.RedisData.NS(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				if !ns.Empty() {
					zap.L().Debug(
						"delegation",
						zap.Uint16("id", context.Req.Id),
					)
					ds, err := h.RedisData.DS(context.zone.Name, location)
					if err != nil {
						context.Res = dns.RcodeServerFailure
						break loop
					}
					if ds.Empty() {
						addNSec(context, currentQName, dns.TypeDS)
					} else {
						if context.QType() == dns.TypeDS {
							context.Answer = append(context.Answer, ds.Value(currentQName)...)
							context.Res = dns.RcodeSuccess
							break loop
						}
						context.Authority = append(context.Authority, ds.Value(currentQName)...)
					}
					context.Authority = append(context.Authority, ns.Value(currentQName)...)
					for _, data := range ns.Data {
						glueLocation, match := context.zone.FindLocation(data.Host)
						if match != types.NoMatch {
							glueA, err := h.RedisData.A(context.zone.Name, glueLocation)
							// XXX : should we return with RcodeServerFailure?
							if err == nil {
								ips := h.filter(context.SourceIp, glueA)
								context.Additional = append(context.Additional, generateA(data.Host, glueA.Ttl(), ips)...)
							}
							glueAAAA, err := h.RedisData.AAAA(context.zone.Name, glueLocation)
							if err == nil {
								ips := h.filter(context.SourceIp, glueAAAA)
								context.Additional = append(context.Additional, generateAAAA(data.Host, glueAAAA.Ttl(), ips)...)
							}
						}
					}
					context.Res = dns.RcodeSuccess
					break loop
				}
			}

			zap.L().Debug(
				"final location",
				zap.Uint16("id", context.Req.Id),
				zap.String("qname", currentQName),
			)
			if cnameFlattening {
				currentQName = context.RawName()
			}
			var answer []dns.RR
			switch context.QType() {
			case dns.TypeA:
				var ips []net.IP
				var ttl uint32
				a, err := h.RedisData.A(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				aname, err := h.RedisData.ANAME(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				if a.Empty() && !aname.Empty() {
					ips, context.Res, ttl = h.findANAME(context, aname.Location, dns.TypeA)
				} else {
					ttl = a.Ttl()
					ips = h.filter(context.SourceIp, a)
				}
				answer = generateA(currentQName, ttl, ips)
			case dns.TypeAAAA:
				var ips []net.IP
				var ttl uint32
				aaaa, err := h.RedisData.AAAA(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				aname, err := h.RedisData.ANAME(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				if aaaa.Empty() && !aname.Empty() {
					ips, context.Res, ttl = h.findANAME(context, aname.Location, dns.TypeAAAA)
				} else {
					ttl = aaaa.Ttl()
					ips = h.filter(context.SourceIp, aaaa)
				}
				answer = generateAAAA(currentQName, ttl, ips)
			case dns.TypeCNAME:
				cname, err := h.RedisData.CNAME(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				answer = cname.Value(currentQName)
			case dns.TypeTXT:
				txt, err := h.RedisData.TXT(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				answer = txt.Value(currentQName)
			case dns.TypeNS:
				ns, err := h.RedisData.NS(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				answer = ns.Value(currentQName)
			case dns.TypeMX:
				mx, err := h.RedisData.MX(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				answer = mx.Value(currentQName)
			case dns.TypeSRV:
				srv, err := h.RedisData.SRV(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				answer = srv.Value(currentQName)
			case dns.TypeCAA:
				// TODO: handle findCAA error response
				caa := h.findCAA(context, currentQName)
				if caa != nil {
					answer = caa.Value(currentQName)
				}
			case dns.TypePTR:
				ptr, err := h.RedisData.PTR(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				answer = ptr.Value(currentQName)
			case dns.TypeTLSA:

				tlsa, err := h.RedisData.TLSA(context.zone.Name, location)
				if err != nil {
					context.Res = dns.RcodeServerFailure
					break loop
				}
				answer = tlsa.Value(currentQName)
			case dns.TypeSOA:
				answer = []dns.RR{context.zone.Config.SOA.Data}
			case dns.TypeDNSKEY:
				if context.zone.Config.DnsSec {
					answer = []dns.RR{context.zone.ZSK.DnsKey, context.zone.KSK.DnsKey}
				}
			case dns.TypeDS:
				answer = []dns.RR{}
			default:
				context.Answer = []dns.RR{}
				context.Authority = []dns.RR{context.zone.Config.SOA.Data}
				context.Res = dns.RcodeSuccess
				break loop
			}
			context.Answer = append(context.Answer, answer...)
			if len(answer) == 0 {
				addNSec(context, currentQName, context.QType())
				if context.Res == dns.RcodeSuccess {
					context.Authority = append(context.Authority, context.zone.Config.SOA.Data)
				}
			}
			break loop
		}
	}

	applyDnssec(context)

	h.response(context)
	zap.L().Debug(
		"end handle request",
		zap.Uint16("id", context.Req.Id),
		zap.String("query", context.RawName()),
		zap.String("type", context.Type()),
	)
}

func (h *DnsRequestHandler) filter(sourceIp net.IP, rrset *types.IP_RRSet) []net.IP {
	mask := make([]int, len(rrset.Data))
	// TODO: filterHealthCheck in redisStat
	//mask = h.healthcheck.FilterHealthcheck(name, rrset, mask)
	switch rrset.FilterConfig.GeoFilter {
	case "asn":
		mask, _ = geotools.GetSameASN(h.geoip, sourceIp, rrset.Data, mask)
	case "country":
		mask, _ = geotools.GetSameCountry(h.geoip, sourceIp, rrset.Data, mask)
	case "asn+country":
		mask, _ = geotools.GetSameASN(h.geoip, sourceIp, rrset.Data, mask)
		mask, _ = geotools.GetSameCountry(h.geoip, sourceIp, rrset.Data, mask)
	case "location":
		mask, _ = geotools.GetMinimumDistance(h.geoip, sourceIp, rrset.Data, mask)
	default:
	}

	return orderIps(rrset, mask)
}

func (h *DnsRequestHandler) logRequest(state *RequestContext) {
	h.requestLogger.Info("query",
		zap.String("domain.id", state.DomainUid),
		zap.String("qname", state.Name()),
		zap.String("qtype", state.Type()),
		zap.String("source.ip", state.SourceIp.String()),
		zap.String("source.subnet", state.SourceSubnet),
		zap.String("source.country", state.SourceCountry),
		zap.Uint("source.asn", state.SourceASN),
		zap.Duration("process_time", time.Since(state.StartTime)),
		zap.Int("response_code", state.Res),
	)
}

func generateA(name string, ttl uint32, ips []net.IP) (answers []dns.RR) {
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA,
			Class: dns.ClassINET, Ttl: ttl}
		r.A = ip
		answers = append(answers, r)
	}
	return
}

func generateAAAA(name string, ttl uint32, ips []net.IP) (answers []dns.RR) {
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		r := new(dns.AAAA)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA,
			Class: dns.ClassINET, Ttl: ttl}
		r.AAAA = ip
		answers = append(answers, r)
	}
	return
}

func orderIps(rrset *types.IP_RRSet, mask []int) []net.IP {
	sum := 0
	count := 0
	for i, x := range mask {
		if x == types.IpMaskWhite {
			count++
			sum += rrset.Data[i].Weight
		}
	}
	result := make([]net.IP, 0, count)
	if count == 0 {
		return result
	}

	index := -1
	if rrset.FilterConfig.Order == "weighted" && sum > 0 {
		s := time.Now().Nanosecond() % sum
		for i, x := range mask {
			if x == types.IpMaskWhite {
				// skip Ips with 0 weight
				s -= rrset.Data[i].Weight
				if s < 0 {
					index = i
					break
				}
			}
		}
	} else if rrset.FilterConfig.Order == "rr" || (rrset.FilterConfig.Order == "weighted" && sum == 0) {
		r := time.Now().Nanosecond() % count
		for i, x := range mask {
			if x == types.IpMaskWhite {
				if r == 0 {
					index = i
					break
				}
				r--
			}
		}
	} else {
		for i, x := range mask {
			if x == types.IpMaskWhite {
				index = i
				break
			}
		}
	}

	if index == -1 {
		return result
	}

	if rrset.FilterConfig.Count == "single" {
		result = append(result, rrset.Data[index].Ip)
		return result
	} else {
		for i := index; i < len(mask); i++ {
			if mask[i] == types.IpMaskWhite {
				result = append(result, rrset.Data[i].Ip)
			}
		}
		for i := 0; i < index; i++ {
			if mask[i] == types.IpMaskWhite {
				result = append(result, rrset.Data[i].Ip)
			}
		}
		return result
	}
}

func (h *DnsRequestHandler) findCAA(context *RequestContext, query string) *types.CAA_RRSet {
	zone := context.zone
	currentLocation, _ := zone.FindLocation(query)
	currentCAA, err := h.RedisData.CAA(zone.Name, currentLocation)
	if err == nil && !currentCAA.Empty() {
		return currentCAA
	}
	for {
		zap.L().Debug("caa location", zap.String("location", currentLocation))
		splits := strings.SplitAfterN(currentLocation, ".", 2)
		if len(splits) != 2 {
			break
		}
		var match int
		currentLocation, match = zone.FindLocation(splits[1])
		if match != types.ExactMatch && match != types.WildCardMatch {
			currentLocation = splits[1]
			continue
		}
		currentCAA, err := h.RedisData.CAA(zone.Name, currentLocation)
		if err != nil {
			currentLocation = splits[1]
			continue
		}
		if !currentCAA.Empty() {
			return currentCAA
		}
	}
	currentCAA, err = h.RedisData.CAA(zone.Name, "@")
	if err != nil {
		return nil
	}
	return currentCAA
}

func (h *DnsRequestHandler) findANAME(context *RequestContext, aname string, qtype uint16) ([]net.IP, int, uint32) {
	zap.L().Debug("finding aname")
	currentQName := aname
	loopCount := 0
	for {
		if loopCount > 10 {
			zap.L().Error(
				"ANAME loop in request",
				zap.String("query", context.RawName()),
				zap.String("type", context.Type()),
			)
			return []net.IP{}, dns.RcodeServerFailure, 0
		}
		loopCount++

		zoneName := h.RedisData.FindZone(currentQName)
		zap.L().Debug(
			"find zone",
			zap.String("zone", zoneName),
			zap.String("qname", currentQName),
		)
		if zoneName == "" || zoneName != context.zone.Name {
			zap.L().Debug("non-authoritative zone, using upstream")
			upstreamAnswers, upstreamRes := h.upstream.Query(currentQName, qtype)
			if upstreamRes == dns.RcodeSuccess {
				var ips []net.IP
				var upstreamTtl uint32
				if len(upstreamAnswers) > 0 {
					upstreamTtl = upstreamAnswers[0].Header().Ttl
				}
				for _, r := range upstreamAnswers {
					if qtype == dns.TypeA {
						if a, ok := r.(*dns.A); ok {
							ips = append(ips, a.A)
						}
					} else {
						if aaaa, ok := r.(*dns.AAAA); ok {
							ips = append(ips, aaaa.AAAA)
						}
					}
				}
				return ips, upstreamRes, upstreamTtl
			} else {
				return []net.IP{}, dns.RcodeServerFailure, 0
			}
		}

		location, matchType := context.zone.FindLocation(currentQName)
		if matchType == types.NoMatch {
			zap.L().Debug(
				"location not found",
				zap.String("qname", currentQName),
			)
			return []net.IP{}, dns.RcodeServerFailure, 0
		}

		cname, err := h.RedisData.CNAME(context.zone.Name, location)
		if err != nil {
			return []net.IP{}, dns.RcodeServerFailure, 0
		}
		if !cname.Empty() {
			zap.L().Debug("cname")
			currentQName = cname.Host
			continue
		}

		if qtype == dns.TypeA {
			a, err := h.RedisData.A(context.zone.Name, location)
			if err != nil {
				return []net.IP{}, dns.RcodeServerFailure, 0
			}
			if !a.Empty() {
				zap.L().Debug("found a")
				return h.filter(context.SourceIp, a), dns.RcodeSuccess, a.TtlValue
			}
		} else if qtype == dns.TypeAAAA {
			aaaa, err := h.RedisData.AAAA(context.zone.Name, location)
			if err != nil {
				return []net.IP{}, dns.RcodeServerFailure, 0
			}
			if !aaaa.Empty() {
				zap.L().Debug("found aaaa")
				return h.filter(context.SourceIp, aaaa), dns.RcodeSuccess, aaaa.TtlValue
			}
		}

		aname, err := h.RedisData.ANAME(context.zone.Name, location)
		if err != nil {
			return []net.IP{}, dns.RcodeServerFailure, 0
		}
		if !aname.Empty() {
			zap.L().Debug("aname")
			currentQName = aname.Location
			continue
		}

		return []net.IP{}, dns.RcodeSuccess, 0
	}
}

func addNSec(context *RequestContext, name string, qtype uint16) {
	if !context.dnssec {
		return
	}
	var bitmap []uint16
	if name == context.zone.Name {
		context.Res = dns.RcodeSuccess
		bitmap = dnssec.FilterNsecBitmap(qtype, dnssec.NsecBitmapAppex)
	} else {
		if context.Res == dns.RcodeNameError {
			context.Res = dns.RcodeSuccess
			bitmap = dnssec.NsecBitmapNameError
		} else {
			if qtype == dns.TypeDS {
				bitmap = dnssec.FilterNsecBitmap(qtype, dnssec.NsecBitmapSubDelegation)
			} else {
				bitmap = dnssec.FilterNsecBitmap(qtype, dnssec.NsecBitmapZone)
			}
		}
	}

	nsec := &dns.NSEC{
		Hdr:        dns.RR_Header{Name: name, Rrtype: dns.TypeNSEC, Class: dns.ClassINET, Ttl: context.zone.Config.SOA.MinTtl},
		NextDomain: "\\000." + name,
		TypeBitMap: bitmap,
	}
	context.Authority = append(context.Authority, nsec)
}

func applyDnssec(context *RequestContext) {
	if !context.dnssec {
		return
	}
	context.Answer = dnssec.SignResponse(context.Answer, context.RawName(), context.zone)
	context.Authority = dnssec.SignResponse(context.Authority, context.RawName(), context.zone)
	// context.Additional = Sign(context.Additional, context.RawName(), zone)

}

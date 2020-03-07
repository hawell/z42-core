package handler

import (
	"github.com/hawell/redins/dnssec"
	"github.com/hawell/redins/handler/logformat"
	"github.com/hawell/redins/redis"
	"github.com/hawell/redins/types"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hawell/logger"
	"github.com/miekg/dns"
)

type DnsRequestHandler struct {
	Config    *DnsRequestHandlerConfig
	RedisData *redis.DataHandler
	Logger    *logger.EventLogger
	geoip     *GeoIp
	upstream  *Upstream
	quit      chan struct{}
	quitWG    sync.WaitGroup
	logQueue  chan map[string]interface{}
}

type DnsRequestHandlerConfig struct {
	Upstream          []UpstreamConfig `json:"upstream"`
	GeoIp             GeoIpConfig      `json:"geoip"`
	MaxTtl            int              `json:"max_ttl"`
	LogSourceLocation bool             `json:"log_source_location"`
	Log               logger.LogConfig `json:"log"`
}

func NewHandler(config *DnsRequestHandlerConfig, redisData *redis.DataHandler) *DnsRequestHandler {
	h := &DnsRequestHandler{
		Config:    config,
		RedisData: redisData,
	}

	getFormatter := func(name string) logrus.Formatter {
		switch name {
		case "capnp_request":
			return &logformat.CapnpRequestLogFormatter{}
		case "json":
			return &logrus.JSONFormatter{TimestampFormat: h.Config.Log.TimeFormat}
		case "text":
			return &logrus.TextFormatter{TimestampFormat: h.Config.Log.TimeFormat}
		default:
			return &logrus.TextFormatter{TimestampFormat: h.Config.Log.TimeFormat}
		}
	}

	h.logQueue = make(chan map[string]interface{}, 1000)
	go func() {
		h.quitWG.Add(1)
		for {
			select {
			case <-h.quit:
				h.quitWG.Done()
				return
			case data := <-h.logQueue:
				h.Logger.Log(data, "dns request")
			}
		}
	}()
	h.Logger = logger.NewLogger(&config.Log, getFormatter)
	h.geoip = NewGeoIp(&config.GeoIp)
	h.upstream = NewUpstream(config.Upstream)
	h.quit = make(chan struct{})

	return h
}

func (h *DnsRequestHandler) ShutDown() {
	// logger.Default.Debug("handler : stopping")
	close(h.logQueue)
	close(h.quit)
	h.quitWG.Wait()
	// logger.Default.Debug("handler : stopped")
}

func (h *DnsRequestHandler) Response(context *RequestContext) {
	h.LogRequest(context)
	context.Response()
}

func (h *DnsRequestHandler) HandleRequest(context *RequestContext) {
	// logger.Default.Debugf("[%d] start handle request - name : %s, type : %s", context.Req.Id, context.RawName(), context.Type())
	if h.Config.LogSourceLocation {
		sourceIP := context.SourceIp
		sourceCountry, _ := h.geoip.GetCountry(sourceIP)
		context.LogData["source_country"] = sourceCountry
		sourceASN, _ := h.geoip.GetASN(sourceIP)
		context.LogData["source_asn"] = sourceASN
	}

	zoneName := h.RedisData.FindZone(context.RawName())
	if zoneName == "" {
		context.Res = dns.RcodeNotAuth
		h.Response(context)
		return
	}
	// logger.Default.Debugf("[%d] zone name : %s", context.Req.Id, zoneName)

	zone := h.RedisData.GetZone(zoneName)
	if zone == nil {
		context.Res = dns.RcodeServerFailure
		h.Response(context)
		return
	}
	context.LogData["domain_uuid"] = zone.Config.DomainId

	context.dnssec = context.Do() && context.Auth && zone.Config.DnsSec
	cnameFlattening := context.dnssec || zone.Config.CnameFlattening

	loopCount := 0
	currentQName := context.RawName()
	currentRecord := &types.Record{}
loop:
	for {
		if loopCount > 10 {
			logger.Default.Errorf("CNAME loop in request %s->%s", context.RawName(), context.Type())
			context.Answer = []dns.RR{}
			context.Res = dns.RcodeServerFailure
			break loop
		}
		loopCount++

		if h.RedisData.FindZone(currentQName) != zoneName {
			// logger.Default.Debugf("[%d] out of zone - qname : %s, zone : %s", context.Req.Id, currentQName, zoneName)
			context.Res = dns.RcodeSuccess
			if len(context.Answer) == 0 {
				NSec(context, context.RawName(), context.QType(), zone)
			}
			break loop
		}

		location, match := zone.FindLocation(currentQName)
		switch match {
		case types.NoMatch:
			// logger.Default.Debugf("[%d] no location matched for %s in %s", context.Req.Id, currentQName, zoneName)
			context.Authority = []dns.RR{zone.Config.SOA.Data}
			context.Res = dns.RcodeNameError
			NSec(context, currentQName, dns.TypeNone, zone)
			break loop

		case types.EmptyNonterminalMatch:
			// logger.Default.Debugf("[%d] empty nonterminal match: %s", context.Req.Id)
			context.Authority = []dns.RR{zone.Config.SOA.Data}
			context.Res = dns.RcodeSuccess
			NSec(context, currentQName, dns.TypeNone, zone)
			break loop

		case types.CEMatch:
			// logger.Default.Debugf("[%d] ce match: %s -> %s", context.Req.Id, currentQName, location)
			currentRecord = h.RedisData.GetLocation(location, zone)
			if currentRecord == nil {
				context.Res = dns.RcodeServerFailure
				break loop
			}
			if len(currentRecord.NS.Data) > 0 && currentRecord.Name != zone.Name {
				// logger.Default.Debugf("[%d] delegation", context.Req.Id)
				context.Authority = append(context.Authority, h.NS(currentRecord.Name, currentRecord)...)
				ds := h.DS(currentRecord.Name, currentRecord)
				if len(ds) == 0 {
					NSec(context, currentRecord.Name, dns.TypeDS, zone)
				}
				context.Authority = append(context.Authority, ds...)
				for _, ns := range currentRecord.NS.Data {
					glueLocation, match := zone.FindLocation(ns.Host)
					if match != types.NoMatch {
						glueRecord := h.RedisData.GetLocation(glueLocation, zone)
						// XXX : should we return with RcodeServerFailure?
						if glueRecord != nil {
							ips := h.Filter(glueRecord.Name, context.SourceIp, &glueRecord.A)
							context.Additional = append(context.Additional, h.A(ns.Host, glueRecord, ips)...)
							ips = h.Filter(glueRecord.Name, context.SourceIp, &glueRecord.AAAA)
							context.Additional = append(context.Additional, h.AAAA(ns.Host, glueRecord, ips)...)
						}
					}
				}
				context.Res = dns.RcodeSuccess
				break loop
			} else {
				context.Authority = []dns.RR{zone.Config.SOA.Data}
				context.Res = dns.RcodeNameError
				NSec(context, currentQName, context.QType(), zone)
				break loop
			}

		case types.WildCardMatch:
			// logger.Default.Debugf("[%d] wildcard match: %s", context.Req.Id, currentRecord.Name)
			fallthrough

		case types.ExactMatch:
			// logger.Default.Debugf("[%d] loading location %s", context.Req.Id, location)
			currentRecord = h.RedisData.GetLocation(location, zone)
			if currentRecord == nil {
				context.Res = dns.RcodeServerFailure
				break loop
			}
			if currentRecord.CNAME != nil && context.QType() != dns.TypeCNAME {
				// logger.Default.Debugf("[%d] cname chain %s -> %s", context.Req.Id, currentQName, currentRecord.CNAME.Host)
				if !cnameFlattening {
					context.Answer = append(context.Answer, h.CNAME(currentQName, currentRecord)...)
				} else if h.RedisData.FindZone(currentRecord.CNAME.Host) != zoneName {
					context.Answer = append(context.Answer, h.CNAME(context.RawName(), currentRecord)...)
					context.Res = dns.RcodeSuccess
					break loop
				}
				currentQName = dns.Fqdn(currentRecord.CNAME.Host)
				continue
			}
			if len(currentRecord.NS.Data) > 0 && currentQName != zone.Name {
				// logger.Default.Debugf("[%d] delegation", context.Req.Id)
				ds := h.DS(currentQName, currentRecord)
				if len(ds) == 0 {
					NSec(context, currentRecord.Name, dns.TypeDS, zone)
				}
				if context.QType() == dns.TypeDS {
					context.Answer = append(context.Answer, ds...)
					context.Res = dns.RcodeSuccess
					break loop
				}
				context.Authority = append(context.Authority, h.NS(currentQName, currentRecord)...)
				context.Authority = append(context.Authority, ds...)
				for _, ns := range currentRecord.NS.Data {
					glueLocation, match := zone.FindLocation(ns.Host)
					if match != types.NoMatch {
						glueRecord := h.RedisData.GetLocation(glueLocation, zone)
						// XXX : should we return with RcodeServerFailure?
						if glueRecord != nil {
							ips := h.Filter(glueRecord.Name, context.SourceIp, &glueRecord.A)
							context.Additional = append(context.Additional, h.A(ns.Host, glueRecord, ips)...)
							ips = h.Filter(glueRecord.Name, context.SourceIp, &glueRecord.AAAA)
							context.Additional = append(context.Additional, h.AAAA(ns.Host, glueRecord, ips)...)
						}
					}
				}
				context.Res = dns.RcodeSuccess
				break loop
			}

			// logger.Default.Debugf("[%d] final location : %s", context.Req.Id, currentQName)
			if cnameFlattening {
				currentQName = context.RawName()
			}
			var answer []dns.RR
			switch context.QType() {
			case dns.TypeA:
				var ips []net.IP
				var ttl uint32
				if len(currentRecord.A.Data) == 0 && currentRecord.ANAME != nil {
					ips, context.Res, ttl = h.FindANAME(context, currentRecord.ANAME.Location, dns.TypeA)
					currentRecord.A.Ttl = ttl
				} else {
					ips = h.Filter(currentRecord.Name, context.SourceIp, &currentRecord.A)
				}
				answer = h.A(currentQName, currentRecord, ips)
			case dns.TypeAAAA:
				var ips []net.IP
				var ttl uint32
				if len(currentRecord.AAAA.Data) == 0 && currentRecord.ANAME != nil {
					ips, context.Res, ttl = h.FindANAME(context, currentRecord.ANAME.Location, dns.TypeAAAA)
					currentRecord.AAAA.Ttl = ttl
				} else {
					ips = h.Filter(currentRecord.Name, context.SourceIp, &currentRecord.AAAA)
				}
				answer = h.AAAA(currentQName, currentRecord, ips)
			case dns.TypeCNAME:
				answer = h.CNAME(currentQName, currentRecord)
			case dns.TypeTXT:
				answer = h.TXT(currentQName, currentRecord)
			case dns.TypeNS:
				answer = h.NS(currentQName, currentRecord)
			case dns.TypeMX:
				answer = h.MX(currentQName, currentRecord)
			case dns.TypeSRV:
				answer = h.SRV(currentQName, currentRecord)
			case dns.TypeCAA:
				// TODO: handle FindCAA error response
				caaRecord := h.FindCAA(currentRecord)
				if caaRecord != nil {
					answer = h.CAA(currentQName, caaRecord)
				}
			case dns.TypePTR:
				answer = h.PTR(currentQName, currentRecord)
			case dns.TypeTLSA:
				answer = h.TLSA(currentQName, currentRecord)
			case dns.TypeSOA:
				answer = []dns.RR{zone.Config.SOA.Data}
			case dns.TypeDNSKEY:
				if zone.Config.DnsSec {
					answer = []dns.RR{zone.ZSK.DnsKey, zone.KSK.DnsKey}
				}
			case dns.TypeDS:
				answer = []dns.RR{}
			default:
				context.Answer = []dns.RR{}
				context.Authority = []dns.RR{zone.Config.SOA.Data}
				context.Res = dns.RcodeNotImplemented
				break loop
			}
			context.Answer = append(context.Answer, answer...)
			if len(answer) == 0 {
				NSec(context, currentQName, context.QType(), zone)
				if context.Res == dns.RcodeSuccess {
					context.Authority = append(context.Authority, zone.Config.SOA.Data)
				}
			}
			break loop
		}
	}

	ApplyDnssec(context, zone)

	h.Response(context)
	// logger.Default.Debugf("[%d] end handle request - name : %s, type : %s", context.Req.Id, context.RawName(), context.Type())
}

func (h *DnsRequestHandler) Filter(name string, sourceIp net.IP, rrset *types.IP_RRSet) []net.IP {
	mask := make([]int, len(rrset.Data))
	// TODO: filterHealthCheck in redisStat
	//mask = h.healthcheck.FilterHealthcheck(name, rrset, mask)
	switch rrset.FilterConfig.GeoFilter {
	case "asn":
		mask = h.geoip.GetSameASN(sourceIp, rrset.Data, mask)
	case "country":
		mask = h.geoip.GetSameCountry(sourceIp, rrset.Data, mask)
	case "asn+country":
		mask = h.geoip.GetSameASN(sourceIp, rrset.Data, mask)
		mask = h.geoip.GetSameCountry(sourceIp, rrset.Data, mask)
	case "location":
		mask = h.geoip.GetMinimumDistance(sourceIp, rrset.Data, mask)
	default:
	}

	return OrderIps(rrset, mask)
}

func (h *DnsRequestHandler) LogRequest(state *RequestContext) {
	state.LogData["process_time"] = time.Since(state.StartTime).Nanoseconds() / 1000000
	state.LogData["response_code"] = state.Res
	state.LogData["log_type"] = "request"
	select {
	case h.logQueue <- state.LogData:
	default:
		logger.Default.Warning("log queue is full")
	}
}

func (h *DnsRequestHandler) A(name string, record *types.Record, ips []net.IP) (answers []dns.RR) {
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA,
			Class: dns.ClassINET, Ttl: h.getTtl(record.A.Ttl)}
		r.A = ip
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) AAAA(name string, record *types.Record, ips []net.IP) (answers []dns.RR) {
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		r := new(dns.AAAA)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA,
			Class: dns.ClassINET, Ttl: h.getTtl(record.AAAA.Ttl)}
		r.AAAA = ip
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) CNAME(name string, record *types.Record) (answers []dns.RR) {
	if record.CNAME == nil {
		return
	}
	r := new(dns.CNAME)
	r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME,
		Class: dns.ClassINET, Ttl: h.getTtl(record.CNAME.Ttl)}
	r.Target = dns.Fqdn(record.CNAME.Host)
	answers = append(answers, r)
	return
}

func (h *DnsRequestHandler) TXT(name string, record *types.Record) (answers []dns.RR) {
	for _, txt := range record.TXT.Data {
		if len(txt.Text) == 0 {
			continue
		}
		r := new(dns.TXT)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeTXT,
			Class: dns.ClassINET, Ttl: h.getTtl(record.TXT.Ttl)}
		r.Txt = split255(txt.Text)
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) NS(name string, record *types.Record) (answers []dns.RR) {
	for _, ns := range record.NS.Data {
		if len(ns.Host) == 0 {
			continue
		}
		r := new(dns.NS)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeNS,
			Class: dns.ClassINET, Ttl: h.getTtl(record.NS.Ttl)}
		r.Ns = dns.Fqdn(ns.Host)
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) MX(name string, record *types.Record) (answers []dns.RR) {
	for _, mx := range record.MX.Data {
		if len(mx.Host) == 0 {
			continue
		}
		r := new(dns.MX)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeMX,
			Class: dns.ClassINET, Ttl: h.getTtl(record.MX.Ttl)}
		r.Mx = dns.Fqdn(mx.Host)
		r.Preference = mx.Preference
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) SRV(name string, record *types.Record) (answers []dns.RR) {
	for _, srv := range record.SRV.Data {
		if len(srv.Target) == 0 {
			continue
		}
		r := new(dns.SRV)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeSRV,
			Class: dns.ClassINET, Ttl: h.getTtl(record.SRV.Ttl)}
		r.Target = dns.Fqdn(srv.Target)
		r.Weight = srv.Weight
		r.Port = srv.Port
		r.Priority = srv.Priority
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) CAA(name string, record *types.Record) (answers []dns.RR) {
	for _, caa := range record.CAA.Data {
		r := new(dns.CAA)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeCAA,
			Class: dns.ClassINET, Ttl: h.getTtl(record.CAA.Ttl)}
		r.Value = caa.Value
		r.Flag = caa.Flag
		r.Tag = caa.Tag
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) PTR(name string, record *types.Record) (answers []dns.RR) {
	if record.PTR == nil {
		return
	}
	r := new(dns.PTR)
	r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypePTR,
		Class: dns.ClassINET, Ttl: h.getTtl(record.PTR.Ttl)}
	r.Ptr = dns.Fqdn(record.PTR.Domain)
	answers = append(answers, r)
	return
}

func (h *DnsRequestHandler) TLSA(name string, record *types.Record) (answers []dns.RR) {
	for _, tlsa := range record.TLSA.Data {
		r := new(dns.TLSA)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeTLSA,
			Class: dns.ClassNONE, Ttl: h.getTtl(record.TLSA.Ttl)}
		r.Usage = tlsa.Usage
		r.Selector = tlsa.Selector
		r.MatchingType = tlsa.MatchingType
		r.Certificate = tlsa.Certificate
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) DS(name string, record *types.Record) (answers []dns.RR) {
	for _, ds := range record.DS.Data {
		r := new(dns.DS)
		r.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeDS,
			Class: dns.ClassINET, Ttl: h.getTtl(record.DS.Ttl)}
		r.KeyTag = ds.KeyTag
		r.Algorithm = ds.Algorithm
		r.DigestType = ds.DigestType
		r.Digest = ds.Digest
		answers = append(answers, r)
	}
	return
}

func (h *DnsRequestHandler) getTtl(ttl uint32) uint32 {
	maxTtl := uint32(h.Config.MaxTtl)
	if ttl == 0 {
		return maxTtl
	}
	if maxTtl == 0 {
		return ttl
	}
	if ttl > maxTtl {
		return maxTtl
	}
	return ttl
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

func OrderIps(rrset *types.IP_RRSet, mask []int) []net.IP {
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

func (h *DnsRequestHandler) FindCAA(record *types.Record) *types.Record {
	zone := record.Zone
	currentRecord := record
	currentLocation := strings.TrimSuffix(currentRecord.Name, "."+zone.Name)
	if len(currentRecord.CAA.Data) != 0 {
		return currentRecord
	}
	for {
		// logger.Default.Debug("location : ", currentLocation)
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
		currentRecord = h.RedisData.GetLocation(currentLocation, zone)
		if currentRecord == nil {
			currentLocation = splits[1]
			continue
		}
		if len(currentRecord.CAA.Data) != 0 {
			return currentRecord
		}
	}
	currentRecord = h.RedisData.GetLocation(zone.Name, zone)
	if currentRecord == nil {
		return nil
	}
	if len(currentRecord.CAA.Data) != 0 {
		return currentRecord
	}
	return nil
}

func (h *DnsRequestHandler) FindANAME(context *RequestContext, aname string, qtype uint16) ([]net.IP, int, uint32) {
	// logger.Default.Debug("finding aname")
	currentQName := aname
	currentRecord := &types.Record{}
	loopCount := 0
	for {
		if loopCount > 10 {
			logger.Default.Errorf("ANAME loop in request %s->%s", context.RawName(), context.Type())
			return []net.IP{}, dns.RcodeServerFailure, 0
		}
		loopCount++

		zoneName := h.RedisData.FindZone(currentQName)
		// logger.Default.Debug("zone : ", zoneName, " qname : ", currentQName, " record : ", currentRecord.Name)
		if zoneName == "" {
			// logger.Default.Debug("non-authoritative zone, using upstream")
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

		zone := h.RedisData.GetZone(zoneName)
		if zone == nil {
			// logger.Default.Debugf("error loading zone : %s", zoneName)
			return []net.IP{}, dns.RcodeServerFailure, 0
		}
		location, _ := zone.FindLocation(currentQName)
		if location == "" {
			// logger.Default.Debugf("location not found for %s", currentQName)
			return []net.IP{}, dns.RcodeServerFailure, 0
		}

		currentRecord = h.RedisData.GetLocation(location, zone)
		if currentRecord == nil {
			return []net.IP{}, dns.RcodeServerFailure, 0
		}
		if currentRecord.CNAME != nil {
			// logger.Default.Debug("cname")
			currentQName = currentRecord.CNAME.Host
			continue
		}

		if qtype == dns.TypeA && len(currentRecord.A.Data) > 0 {
			// logger.Default.Debug("found a")
			return h.Filter(currentRecord.Name, context.SourceIp, &currentRecord.A), dns.RcodeSuccess, currentRecord.A.Ttl
		} else if qtype == dns.TypeAAAA && len(currentRecord.AAAA.Data) > 0 {
			// logger.Default.Debug("found aaaa")
			return h.Filter(currentRecord.Name, context.SourceIp, &currentRecord.AAAA), dns.RcodeSuccess, currentRecord.AAAA.Ttl
		}

		if currentRecord.ANAME != nil {
			// logger.Default.Debug("aname")
			currentQName = currentRecord.ANAME.Location
			continue
		}

		return []net.IP{}, dns.RcodeSuccess, 0
	}
}

func NSec(context *RequestContext, name string, qtype uint16, zone *types.Zone) {
	if !context.dnssec {
		return
	}
	var bitmap []uint16
	if name == zone.Name {
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
		Hdr:        dns.RR_Header{Name: name, Rrtype: dns.TypeNSEC, Class: dns.ClassINET, Ttl: zone.Config.SOA.MinTtl},
		NextDomain: "\\000." + name,
		TypeBitMap: bitmap,
	}
	context.Authority = append(context.Authority, nsec)
}

func ApplyDnssec(context *RequestContext, zone *types.Zone) {
	if !context.dnssec {
		return
	}
	context.Answer = dnssec.SignResponse(context.Answer, context.RawName(), zone)
	context.Authority = dnssec.SignResponse(context.Authority, context.RawName(), zone)
	// context.Additional = Sign(context.Additional, context.RawName(), zone)

}

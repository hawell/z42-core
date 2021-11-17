package resolver

import (
	"github.com/coredns/coredns/request"
	"github.com/hawell/z42/internal/types"
	"github.com/miekg/dns"
	"net"
	"strings"
	"time"
)

type RequestContext struct {
	request.Request
	StartTime  time.Time
	Auth       bool
	Res        int
	dnssec     bool
	Answer     []dns.RR
	Authority  []dns.RR
	Additional []dns.RR

	DomainUid     string
	SourceIp      net.IP
	SourceSubnet  string
	SourceCountry string
	SourceASN     uint

	name string

	zone *types.Zone
}

func NewRequestContext(w dns.ResponseWriter, r *dns.Msg) *RequestContext {
	context := &RequestContext{
		Request: request.Request{
			Req:  r,
			W:    w,
			Zone: "",
		},
		StartTime: time.Now(),
		Auth:      true,
		Res:       dns.RcodeSuccess,
		dnssec:    false,
		name:      "",
	}
	context.SourceIp = context.sourceIp()
	context.SourceSubnet = context.sourceSubnet()
	return context
}

func (context *RequestContext) sourceIp() net.IP {
	opt := context.Req.IsEdns0()
	if opt != nil && len(opt.Option) != 0 {
		for _, o := range opt.Option {
			switch v := o.(type) {
			case *dns.EDNS0_SUBNET:
				return v.Address
			}
		}
	}
	return net.ParseIP(context.IP())
}

func (context *RequestContext) sourceSubnet() string {
	opt := context.Req.IsEdns0()
	if opt != nil && len(opt.Option) != 0 {
		for _, o := range opt.Option {
			switch o.(type) {
			case *dns.EDNS0_SUBNET:
				return o.String()
			}
		}
	}
	return ""
}

func (context *RequestContext) RawName() string {
	if context.name != "" {
		return context.name
	}
	if context.Req == nil {
		context.name = "."
		return "."
	}
	if len(context.Req.Question) == 0 {
		context.name = "."
		return "."
	}

	context.name = strings.ToLower(context.Req.Question[0].Name)
	return context.name
}

func (context *RequestContext) Response() {
	m := new(dns.Msg)
	m.Authoritative, m.RecursionAvailable, m.Compress = context.Auth, false, true
	m.SetRcode(context.Req, context.Res)
	m.Answer = append(m.Answer, context.Answer...)
	m.Ns = append(m.Ns, context.Authority...)
	m.Extra = append(m.Extra, context.Additional...)

	context.SizeAndDo(m)
	m = context.Scrub(m)
	if err := context.W.WriteMsg(m); err != nil {
		// zap.L().Error("write error", zap.Error(err), zap.String("msg", m.String()))
		_ = context.W.Close()
	}
}

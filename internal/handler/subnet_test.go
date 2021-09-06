package handler

import (
	"github.com/hawell/z42/internal/test"
	"github.com/miekg/dns"
	. "github.com/onsi/gomega"
	"net"
	"testing"
)

func TestSubnet(t *testing.T) {
	RegisterTestingT(t)
	tc := test.Case{
		Qname: "example.com.", Qtype: dns.TypeA,
	}
	sa := "192.168.1.2"
	opt := &dns.OPT{
		Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT, Class: dns.ClassANY, Rdlength: 0, Ttl: 300},
		Option: []dns.EDNS0{
			&dns.EDNS0_SUBNET{
				Address:       net.ParseIP(sa),
				Code:          dns.EDNS0SUBNET,
				Family:        1,
				SourceNetmask: 32,
				SourceScope:   0,
			},
		},
	}
	r := tc.Msg()
	r.Extra = append(r.Extra, opt)

	Expect(r.IsEdns0()).NotTo(BeNil())
	w := test.NewRecorder(&test.ResponseWriter{})
	state := NewRequestContext(w, r)

	subnet := state.SourceSubnet
	Expect(subnet).To(Equal(sa + "/32/0"))
	address := state.SourceIp
	Expect(address.String()).To(Equal(sa))
}

package upstream

import (
	"fmt"
	"github.com/miekg/dns"
	"testing"
	"time"
)

func TestUpstream_Query(t *testing.T) {
	client := &dns.Client{
		Net:     "udp",
		Timeout: 2000 * time.Millisecond,
	}
	m := new(dns.Msg)

	o := new(dns.OPT)
	o.Hdr.Name = "."
	o.Hdr.Rrtype = dns.TypeOPT
	e := new(dns.EDNS0_COOKIE)
	e.Code = dns.EDNS0COOKIE
	e.Cookie = "24a5ac1223344556"
	o.Option = append(o.Option, e)
	m.Extra = []dns.RR{o}
	m.Id = dns.Id()
	m.SetQuestion("cloudflare.com.", dns.TypeA)
	r, _, err := client.Exchange(m, "4.2.2.4:53")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(r.Rcode)
}

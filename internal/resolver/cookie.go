package resolver

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/dchest/siphash"
	"github.com/miekg/dns"
	"net"
	"time"
)

var (
	CookieErrorNoCookie  = errors.New("no cookie")
	CookieErrorBadFormat = errors.New("bad format")
)

type ClientCookie []byte

type ServerCookie struct {
	raw       []byte
	Version   uint8
	Timestamp uint32
	Hash      [8]byte
}

func UnpackServerCookie(input string) *ServerCookie {
	buf, err := hex.DecodeString(input)
	if err != nil {
		return nil
	}
	p := bytes.NewBuffer(buf)
	s := &ServerCookie{raw: buf}
	if err := binary.Read(p, binary.BigEndian, &s.Version); err != nil {
		return nil
	}
	p.Next(3)
	if err := binary.Read(p, binary.BigEndian, &s.Timestamp); err != nil {
		return nil
	}
	if err := binary.Read(p, binary.BigEndian, &s.Hash); err != nil {
		return nil
	}
	return s
}

func (s *ServerCookie) Pack() string {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, s.Version)
	binary.Write(buf, binary.BigEndian, []byte{0, 0, 0})
	binary.Write(buf, binary.BigEndian, s.Timestamp)
	binary.Write(buf, binary.BigEndian, s.Hash)
	return hex.EncodeToString(buf.Bytes())
}

func (s *ServerCookie) hash(cCookie ClientCookie, sourceAddress net.IP, secret []byte) []byte {
	buf := new(bytes.Buffer)
	var ip net.IP
	if ip = sourceAddress.To4(); ip == nil {
		ip = sourceAddress.To16()
	}
	binary.Write(buf, binary.BigEndian, cCookie)
	binary.Write(buf, binary.BigEndian, s.Version)
	binary.Write(buf, binary.BigEndian, []byte{0, 0, 0})
	binary.Write(buf, binary.BigEndian, s.Timestamp)
	binary.Write(buf, binary.BigEndian, ip)
	h := siphash.New(secret)
	h.Write(buf.Bytes())
	return h.Sum(nil)
}

func GetCookie(r *dns.Msg) (ClientCookie, *ServerCookie, error) {
	o := r.IsEdns0()
	if o == nil {
		return nil, nil, CookieErrorNoCookie
	}
	for _, option := range o.Option {
		if option.Option() == dns.EDNS0COOKIE {
			c := option.(*dns.EDNS0_COOKIE)
			l := len(c.Cookie)
			if l == 16 {
				cCookie, err := hex.DecodeString(c.Cookie[:16])
				if err != nil {
					return nil, nil, CookieErrorBadFormat
				}
				return cCookie, nil, nil
			} else if l == 48 {
				cCookie, err := hex.DecodeString(c.Cookie[:16])
				if err != nil {
					return nil, nil, CookieErrorBadFormat
				}
				sCookie := UnpackServerCookie(c.Cookie[16:])
				if sCookie == nil {
					return nil, nil, CookieErrorBadFormat
				}
				return cCookie, sCookie, nil

			} else {
				return nil, nil, CookieErrorBadFormat
			}
		}
	}
	return nil, nil, CookieErrorNoCookie
}

func CheckCookie(context *RequestContext, cCookie ClientCookie, sCookie *ServerCookie, secret []byte) bool {
	expected := sCookie.hash(cCookie, context.SourceIp, secret)
	if !bytes.Equal(expected, sCookie.Hash[:]) {
		return false
	}
	cookieTime := time.Unix(int64(sCookie.Timestamp), 0)
	before := time.Now().Add(-time.Hour)
	after := time.Now().Add(time.Minute * 5)
	if cookieTime.Before(before) || cookieTime.After(after) {
		return false
	}
	return true
}

func CopyCookie(context *RequestContext, cCookie ClientCookie, sCookie *ServerCookie, secret []byte) {
	cookieTime := time.Unix(int64(sCookie.Timestamp), 0)
	if cookieTime.Before(time.Now().Add(-time.Minute * 30)) {
		SetNewCookie(context, cCookie, secret)
		return
	}
	cookie := &dns.EDNS0_COOKIE{
		Code:   dns.EDNS0COOKIE,
		Cookie: hex.EncodeToString(append(cCookie, sCookie.raw...)),
	}
	setCookie(context, cookie)
}

func SetNewCookie(context *RequestContext, cCookie ClientCookie, secret []byte) {
	sCookie := ServerCookie{
		Version:   1,
		Timestamp: uint32(time.Now().Unix()),
	}
	copy(sCookie.Hash[:], sCookie.hash(cCookie, context.SourceIp, secret))

	cookie := &dns.EDNS0_COOKIE{
		Code:   dns.EDNS0COOKIE,
		Cookie: hex.EncodeToString(cCookie) + sCookie.Pack(),
	}
	setCookie(context, cookie)
}

func setCookie(context *RequestContext, cookie *dns.EDNS0_COOKIE) {
	var e *dns.OPT
	for i := len(context.Additional) - 1; i >= 0; i-- {
		if context.Additional[i].Header().Rrtype == dns.TypeOPT {
			e = context.Additional[i].(*dns.OPT)
		}
	}
	if e == nil {
		e = new(dns.OPT)
		e.Hdr.Name = "."
		e.Hdr.Rrtype = dns.TypeOPT
		e.SetUDPSize(4096)
		e.SetDo()
	}
	e.Option = append(e.Option, cookie)
	context.Additional = append(context.Additional, e)
}

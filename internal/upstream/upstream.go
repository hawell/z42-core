package upstream

import (
	"errors"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"strconv"
	"time"

	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
)

type Connection struct {
	client        *dns.Client
	connectionStr string
}

type Upstream struct {
	connections []*Connection
	cache       *cache.Cache
	inflight    *singleflight.Group
}

func NewUpstream(config []Config) *Upstream {
	u := &Upstream{
		inflight: new(singleflight.Group),
	}

	u.cache = cache.New(time.Second*time.Duration(defaultCacheTtl), time.Second*time.Duration(defaultCacheTtl)*10)
	for _, upstreamConfig := range config {
		client := &dns.Client{
			Net:     upstreamConfig.Protocol,
			Timeout: time.Duration(upstreamConfig.Timeout) * time.Millisecond,
		}
		connectionStr := upstreamConfig.Ip + ":" + strconv.Itoa(upstreamConfig.Port)
		connection := &Connection{
			client:        client,
			connectionStr: connectionStr,
		}
		u.connections = append(u.connections, connection)
	}

	return u
}

func (u *Upstream) Query(location string, qtype uint16) ([]dns.RR, int) {
	key := location + ":" + strconv.Itoa(int(qtype))
	res, exp, found := u.cache.GetWithExpiration(key)
	if found {
		records := res.([]dns.RR)
		for _, record := range records {
			record.Header().Ttl = uint32(time.Until(exp).Seconds())
		}
		return records, dns.RcodeSuccess
	}
	answer, err, _ := u.inflight.Do(key, func() (interface{}, error) {
		m := new(dns.Msg)
		m.SetQuestion(location, qtype)
		for _, c := range u.connections {
			r, _, err := c.client.Exchange(m, c.connectionStr)
			if err != nil {
				zap.L().Error(
					"failed to retrieve record from upstream",
					zap.String("location", location),
					zap.String("upstream", c.connectionStr),
					zap.Error(err),
				)
				continue
			}
			if r.Rcode != dns.RcodeSuccess {
				zap.L().Error("upstream error response", zap.String("rcode", dns.RcodeToString[r.Rcode]), zap.String("location", location))
				return r, nil
			}
			if len(r.Answer) == 0 {
				return r, nil
			}
			minTtl := r.Answer[0].Header().Ttl
			for _, record := range r.Answer {
				if record.Header().Ttl < minTtl {
					minTtl = record.Header().Ttl
				}
			}
			u.cache.Set(key, r.Answer, time.Duration(minTtl)*time.Second)
			u.connections[0], c = c, u.connections[0]

			return r, nil
		}
		return nil, errors.New("failed to retrieve data from upstream")
	})
	if err != nil {
		return []dns.RR{}, dns.RcodeServerFailure
	} else {
		msg := answer.(*dns.Msg)
		return msg.Answer, msg.Rcode
	}
}

const (
	defaultCacheTtl = 600
)

package source

import (
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/hiredis"
	"github.com/hawell/z42/tools/query/query"
)

type RedisDumpQueryGenerator struct {
	redisAddress string
	queries      []query.Query
	pos          int
}

func NewRedisDumpQueryGenerator(redisAddress string) *RedisDumpQueryGenerator {
	dh := storage.NewDataHandler(&storage.DataHandlerConfig{
		ZoneCacheSize:      10000,
		ZoneCacheTimeout:   60,
		ZoneReload:         1,
		RecordCacheSize:    1000000,
		RecordCacheTimeout: 60,
		MinTTL:             5,
		MaxTTL:             300,
		Redis: hiredis.Config{
			Suffix:  "_redis2",
			Prefix:  "",
			Address: redisAddress,
			Net:     "tcp",
			DB:      0,
			Connection: hiredis.ConnectionConfig{
				MaxIdleConnections:   10,
				MaxActiveConnections: 10,
				ConnectTimeout:       500,
				ReadTimeout:          500,
				IdleKeepAlive:        30,
				MaxKeepAlive:         0,
				WaitForConnection:    false,
			},
		},
	})
	g := new(RedisDumpQueryGenerator)
	zones := dh.GetZones()
	for _, zone := range zones {
		locations := dh.GetZoneLocations(zone)
		for _, location := range locations {
			qname := ""
			if location == "@" {
				qname = zone
			} else {
				qname = location + "." + zone
			}
			g.queries = append(g.queries, query.Query{QName: qname, QType: "A"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "AAAA"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "CNAME"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "NS"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "MX"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "SRV"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "TXT"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "PTR"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "CAA"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "TLSA"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "SOA"})
			g.queries = append(g.queries, query.Query{QName: qname, QType: "DNSKEY"})
		}
	}
	return g
}

func (g *RedisDumpQueryGenerator) Init() {

}

func (g *RedisDumpQueryGenerator) Count() int {
	return len(g.queries)
}

func (g *RedisDumpQueryGenerator) GetQuery() query.Query {
	q := g.queries[g.pos]
	g.pos++
	return q
}

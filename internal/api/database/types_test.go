package database

import (
	"z42-core/internal/types"
	jsoniter "github.com/json-iterator/go"
	. "github.com/onsi/gomega"
	"net"
	"testing"
)

func TestZoneImport_UnmarshalJSON(t *testing.T) {
	RegisterTestingT(t)
	importData := `
		{
			"name": "zone1.com.",
			"entries": {
				"@": {
					"a": {"ttl":300, "records":[{"ip":"6.5.6.5"}]},
					"ns": {"ttl": 3600, "records": [{"host": "ns1.example.com."}, {"host": "ns2.example.com."}]}
				},
				"www": {
					"a": {"ttl":300, "records":[{"ip":"6.5.6.5"}]},
					"mx": {"ttl":300, "records":[{"host":"mx1.example.ddd.", "preference":10}]}
				}
			}
		}
	`
	var zoneImport ZoneImport
	err := jsoniter.Unmarshal([]byte(importData), &zoneImport)
	Expect(err).To(BeNil())
	Expect(zoneImport).To(Equal(ZoneImport{
		Name: "zone1.com.",
		Entries: map[string]map[string]types.RRSet{
			"@": {
				"a": &types.IP_RRSet{
					GenericRRSet: types.GenericRRSet{TtlValue: 300},
					Data:         []types.IP_RR{{Ip: net.ParseIP("6.5.6.5")}},
				},
				"ns": &types.NS_RRSet{
					GenericRRSet: types.GenericRRSet{TtlValue: 3600},
					Data:         []types.NS_RR{{Host: "ns1.example.com."}, {Host: "ns2.example.com."}},
				},
			},
			"www": {
				"a": &types.IP_RRSet{
					GenericRRSet: types.GenericRRSet{TtlValue: 300},
					Data:         []types.IP_RR{{Ip: net.ParseIP("6.5.6.5")}},
				},
				"mx": &types.MX_RRSet{
					GenericRRSet: types.GenericRRSet{TtlValue: 300},
					Data:         []types.MX_RR{{Host: "mx1.example.ddd.", Preference: 10}},
				},
			},
		},
	}))
}

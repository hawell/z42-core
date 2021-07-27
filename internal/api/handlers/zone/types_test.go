package zone

import (
	"github.com/hawell/z42/internal/types"
	jsoniter "github.com/json-iterator/go"
	. "github.com/onsi/gomega"
	"net"
	"testing"
)

func TestNewRecordSetRequest_UnmarshalJSON(t *testing.T) {
	data := `{"type": "a", "enabled": true, "value": {"ttl":300, "records":[{"ip":"6.5.6.5"}]}}`
	var req NewRecordSetRequest
	err := jsoniter.Unmarshal([]byte(data), &req)
	Expect(err).To(BeNil())
	Expect(req).To(Equal(NewRecordSetRequest{
		Type:    "a",
		Value:   &types.IP_RRSet{
			GenericRRSet:      types.GenericRRSet{
				TtlValue: 300,
			},
			Data:              []types.IP_RR{
				{
					Ip:      net.ParseIP("6.5.6.5"),
				},
			},
		},
		Enabled: true,
	}))
}

package zone

import (
	"errors"
	"github.com/hawell/z42/internal/types"
	jsoniter "github.com/json-iterator/go"
)

type ListRequest struct {
	Start     int    `form:"start,default=0"`
	Count     int    `form:"count,default=100"`
	Ascending bool   `form:"ascending,default=true"`
	Q         string `form:"q,default="`
}

type ListResponseItem struct {
	Name string `json:"id"`
}

type ListResponse []ListResponseItem

type NewZoneRequest struct {
	Name            string          `json:"name" binding:"required"`
	Enabled         bool            `json:"enabled,default:true"`
	Dnssec          bool            `json:"dnssec,default:false"`
	CNameFlattening bool            `json:"cname_flattening,default:false"`
}

type GetZoneResponse struct {
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	Dnssec          bool            `json:"dnssec"`
	CNameFlattening bool            `json:"cname_flattening"`
	SOA             types.SOA_RRSet `json:"soa"`
}

type UpdateZoneRequest struct {
	Enabled         bool            `json:"enabled,default:true"`
	Dnssec          bool            `json:"dnssec,default:false"`
	CNameFlattening bool            `json:"cname_flattening,default:false"`
	SOA             types.SOA_RRSet `json:"soa"`
}

type NewLocationRequest struct {
	Name    string `json:"name" binding:"required"`
	Enabled bool   `json:"enabled,default:true"`
}

type GetLocationResponse struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type UpdateLocationRequest struct {
	Enabled bool `json:"enabled,default=true"`
}

type NewRecordSetRequest struct {
	Type    string      `json:"type" binding:"required"`
	Value   types.RRSet `json:"value" binding:"required"`
	Enabled bool        `json:"enabled,default=true"`
}

func (r *NewRecordSetRequest) UnmarshalJSON(data []byte) error {
	var dat struct {
		Type    string `json:"type"`
		Enabled bool   `json:"enabled,default=true"`
	}
	if err := jsoniter.Unmarshal(data, &dat); err != nil {
		return err
	}
	value := types.TypeToRRSet[dat.Type]
	if value == nil {
		return errors.New("invalid record type")
	}
	val := struct {
		Value types.RRSet `json:"value"`
	}{
		Value: value(),
	}
	if err := jsoniter.Unmarshal(data, &val); err != nil {
		return err
	}
	r.Type = dat.Type
	r.Enabled = dat.Enabled
	r.Value = val.Value
	return nil
}

type GetRecordSetResponse struct {
	Value   types.RRSet `json:"value"`
	Enabled bool        `json:"enabled"`
}

type UpdateRecordSetRequest struct {
	Value   types.RRSet `json:"value" binding:"required"`
	Enabled bool        `json:"enabled,default=true"`
}
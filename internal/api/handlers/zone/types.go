package zone

type ListRequest struct {
	Start int    `form:"start,default=0"`
	Count int    `form:"count,default=100"`
	Q     string `form:"q,default="`
}

type ListResponse []string

type NewZoneRequest struct {
	Name            string `json:"name" binding:"required"`
	Enabled         bool   `json:"enabled,default:true"`
	Dnssec          bool   `json:"dnssec,default:false"`
	CNameFlattening bool   `json:"cname_flattening,default:false"`
}

type GetZoneResponse struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Dnssec          bool   `json:"dnssec"`
	CNameFlattening bool   `json:"cname_flattening"`
}

type UpdateZoneRequest struct {
	Enabled         bool `json:"enabled,default:true"`
	Dnssec          bool `json:"dnssec,default:false"`
	CNameFlattening bool `json:"cname_flattening,default:false"`
}

type NewLocationRequest struct {
	Name    string `json:"name" binding:"required"`
	Enabled bool   `json:"enabled,default:true"`
}

type GetLocationResponse struct {
	Enabled bool `json:"enabled"`
}

type UpdateLocationRequest struct {
	Enabled bool `json:"enabled,default=true"`
}

type NewRecordSetRequest struct {
	Type    string `json:"type" binding:"required"`
	Value   string `json:"value" binding:"required"`
	Enabled bool   `json:"enabled,default=true"`
}

type GetRecordSetResponse struct {
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type UpdateRecordSetRequest struct {
	Value   string `json:"value" binding:"required"`
	Enabled bool   `json:"enabled,,default=true"`
}

package zone

type listRequest struct {
    Start int    `form:"start,default=0"`
    Count int    `form:"count,default=100"`
    Q     string `form:"q,default="`
}

type newZoneRequest struct {
    Name string `json:"name" binding:"required"`
    Enabled bool `json:"enabled,default:true"`
    Dnssec bool `json:"dnssec,default:false"`
    CNameFlattening bool `json:"cname_flattening,default:false"`
}

type getZoneResponse struct {
    Name string `json:"name"`
    Enabled bool `json:"enabled"`
    Dnssec bool `json:"dnssec"`
    CNameFlattening bool `json:"cname_flattening"`
}

type updateZoneRequest struct {
    Enabled bool `json:"enabled,default:true"`
    Dnssec bool `json:"dnssec,default:false"`
    CNameFlattening bool `json:"cname_flattening,default:false"`
}

type newLocationRequest struct {
    Name string `json:"name" binding:"required"`
    Enabled bool `json:"enabled,default:true"`
}

type getLocationResponse struct {
    Enabled bool `json:"enabled"`
}

type updateLocationRequest struct {
    Enabled bool `json:"enabled,default=true"`
}

type newRecordSetRequest struct {
    Type string `json:"type" binding:"required"`
    Value string `json:"value" binding:"required"`
    Enabled bool `json:"enabled,default=true"`
}

type getRecordSetResponse struct {
    Value string `json:"type"`
    Enabled bool `json:"enabled"`
}

type updateRecordSetRequest struct {
    Value string `json:"type" binding:"required"`
    Enabled bool `json:"enabled,,default=true"`
}
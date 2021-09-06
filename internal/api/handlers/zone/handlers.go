package zone

import (
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/pkg/hiredis"
	"go.uber.org/zap"
	"log"
	"net/http"
)

type storage interface {
	AddZone(userId database.ObjectId, z database.NewZone) (database.ObjectId, error)
	GetZones(userId database.ObjectId, start int, count int, q string, ascendingOrder bool) (database.List, error)
	GetZone(userId database.ObjectId, zoneName string) (database.Zone, error)
	UpdateZone(userId database.ObjectId, z database.ZoneUpdate) error
	DeleteZone(userId database.ObjectId, z database.ZoneDelete) error
	AddLocation(userId database.ObjectId, l database.NewLocation) (database.ObjectId, error)
	GetLocations(userId database.ObjectId, zoneName string, start int, count int, q string, ascendingOrder bool) (database.List, error)
	GetLocation(userId database.ObjectId, zoneName string, location string) (database.Location, error)
	UpdateLocation(userId database.ObjectId, l database.LocationUpdate) error
	DeleteLocation(userId database.ObjectId, l database.LocationDelete) error
	AddRecordSet(userId database.ObjectId, r database.NewRecordSet) (database.ObjectId, error)
	GetRecordSets(userId database.ObjectId, zoneName string, location string) (database.List, error)
	GetRecordSet(userId database.ObjectId, zoneName string, location string, recordType string) (database.RecordSet, error)
	UpdateRecordSet(userId database.ObjectId, r database.RecordSetUpdate) error
	DeleteRecordSet(userId database.ObjectId, r database.RecordSetDelete) error
}

type Handler struct {
	db    storage
	redis *hiredis.Redis
}

func New(db storage, redis *hiredis.Redis) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}

const (
	zoneNameKey   = "zone_name"
	locationKey   = "location"
	recordTypeKey = "record_type"
)

func (h *Handler) RegisterHandlers(group *gin.RouterGroup) {
	group.GET("", h.getZones)
	group.POST("", h.addZone)

	group.GET("/:zone_name", h.getZone)
	group.PUT("/:zone_name", h.updateZone)
	group.DELETE("/:zone_name", h.deleteZone)

	group.GET("/:zone_name/locations", h.getLocations)
	group.POST("/:zone_name/locations", h.addLocation)

	group.GET("/:zone_name/locations/:location", h.getLocation)
	group.PUT("/:zone_name/locations/:location", h.updateLocation)
	group.DELETE("/:zone_name/locations/:location", h.deleteLocation)

	group.GET("/:zone_name/locations/:location/rrsets", h.getRecordSets)
	group.POST("/:zone_name/locations/:location/rrsets", h.addRecordSet)

	group.GET("/:zone_name/locations/:location/rrsets/:record_type", h.getRecordSet)
	group.PUT("/:zone_name/locations/:location/rrsets/:record_type", h.updateRecordSet)
	group.DELETE("/:zone_name/locations/:location/rrsets/:record_type", h.deleteRecordSet)
}

func (h *Handler) getZones(c *gin.Context) {
	log.Println(c.Request.URL.String())
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	var req ListRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	zones, err := h.db.GetZones(userId, req.Start, req.Count, req.Q, req.Ascending)
	if err != nil {
		zap.L().Error("DataBase.getZones()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusOK, "successful", zones)
}

func (h *Handler) addZone(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	var z NewZoneRequest
	if err := c.ShouldBindJSON(&z); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.NewZone{
		Name:            z.Name,
		Enabled:         z.Enabled,
		Dnssec:          z.Dnssec,
		CNameFlattening: z.CNameFlattening,
		SOA:             z.SOA,
		NS:              z.NS,
	}
	_, err := h.db.AddZone(userId, model)
	if err != nil {
		zap.L().Error("DataBase.addZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}

	handlers.SuccessfulOperationResponse(c, http.StatusCreated, "created", z.Name)
}

func (h *Handler) getZone(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	z, err := h.db.GetZone(userId, zoneName)
	if err != nil {
		zap.L().Error("DataBase.GetZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}

	resp := GetZoneResponse{
		Name:            z.Name,
		Enabled:         z.Enabled,
		Dnssec:          z.Dnssec,
		CNameFlattening: z.CNameFlattening,
		SOA:             z.SOA,
	}

	handlers.SuccessResponse(c, http.StatusOK, "successful", resp)
}

func (h *Handler) updateZone(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	var req UpdateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	// TODO: set soa.serial
	z := database.ZoneUpdate{
		Name:            zoneName,
		Enabled:         req.Enabled,
		Dnssec:          req.Dnssec,
		CNameFlattening: req.CNameFlattening,
		SOA:             req.SOA,
	}
	if err := h.db.UpdateZone(userId, z); err != nil {
		zap.L().Error("DataBase.updateZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusNoContent, "successful", zoneName)
}

func (h *Handler) deleteZone(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	err := h.db.DeleteZone(userId, database.ZoneDelete{Name: zoneName})
	if err != nil {
		zap.L().Error("DataBase.deleteZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusNoContent, "successful", zoneName)
}

func (h *Handler) getLocations(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	var req ListRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	locations, err := h.db.GetLocations(userId, zoneName, req.Start, req.Count, req.Q, req.Ascending)
	if err != nil {
		zap.L().Error("DataBase.getLocations()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusOK, "successful", locations)
}

func (h *Handler) addLocation(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	var req NewLocationRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.NewLocation{
		ZoneName: zoneName,
		Location: req.Name,
		Enabled:  req.Enabled,
	}
	_, err = h.db.AddLocation(userId, model)
	if err != nil {
		zap.L().Error("DataBase.addLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusCreated, "successful", req.Name)
}

func (h *Handler) getLocation(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	l, err := h.db.GetLocation(userId, zoneName, location)
	if err != nil {
		zap.L().Error("DataBase.getLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	resp := GetLocationResponse{
		Name:    l.Name,
		Enabled: l.Enabled,
	}
	handlers.SuccessResponse(c, http.StatusOK, "successful", resp)
}

func (h *Handler) updateLocation(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.LocationUpdate{
		ZoneName: zoneName,
		Location: location,
		Enabled:  req.Enabled,
	}
	err := h.db.UpdateLocation(userId, model)
	if err != nil {
		zap.L().Error("DataBase.updateLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusNoContent, "successful", location)
}

func (h *Handler) deleteLocation(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	err := h.db.DeleteLocation(userId, database.LocationDelete{ZoneName: zoneName, Location: location})
	if err != nil {
		zap.L().Error("DataBase.deleteLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusNoContent, "successful", location)
}

func (h *Handler) getRecordSets(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	rrsets, err := h.db.GetRecordSets(userId, zoneName, location)
	if err != nil {
		zap.L().Error("DataBase.getRecordSets()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusOK, "successful", rrsets)
}

func (h *Handler) addRecordSet(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	var req NewRecordSetRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.NewRecordSet{
		ZoneName: zoneName,
		Location: location,
		Type:     req.Type,
		Value:    req.Value,
		Enabled:  req.Enabled,
	}
	if !rtypeValid(req.Type) {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid record type")
		return
	}
	_, err = h.db.AddRecordSet(userId, model)
	if err != nil {
		zap.L().Error("DataBase.addRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusCreated, "successful", req.Type)
}

func (h *Handler) getRecordSet(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	recordType := c.Param(recordTypeKey)
	if !rtypeValid(recordType) {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid record type")
		return
	}
	r, err := h.db.GetRecordSet(userId, zoneName, location, recordType)
	if err != nil {
		zap.L().Error("DataBase.getRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	resp := GetRecordSetResponse{
		Enabled: r.Enabled,
		Value:   r.Value,
	}
	handlers.SuccessResponse(c, http.StatusOK, "successful", resp)
}

func (h *Handler) updateRecordSet(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	recordType := c.Param(recordTypeKey)
	if !rtypeValid(recordType) {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid record type")
		return
	}
	value := types.TypeToRRSet[recordType]()
	req := UpdateRecordSetRequest{
		Value: value,
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.RecordSetUpdate{
		ZoneName: zoneName,
		Location: location,
		Type:     recordType,
		Value:    req.Value,
		Enabled:  req.Enabled,
	}
	err := h.db.UpdateRecordSet(userId, model)
	if err != nil {
		zap.L().Error("DataBase.updateRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusNoContent, "successful", recordType)
}

func (h *Handler) deleteRecordSet(c *gin.Context) {
	userId := extractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	zoneName := c.Param(zoneNameKey)
	if zoneName == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param(locationKey)
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	recordType := c.Param(recordTypeKey)
	if recordType == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "record missing")
		return
	}
	err := h.db.DeleteRecordSet(userId, database.RecordSetDelete{ZoneName: zoneName, Location: location, Type: recordType})
	if err != nil {
		zap.L().Error("DataBase.deleteRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusNoContent, "successful", recordType)
}

func extractUser(c *gin.Context) database.ObjectId {
	user, _ := c.Get(handlers.IdentityKey)
	return user.(*handlers.IdentityData).Id
}

func rtypeValid(rtype string) bool {
	_, ok := types.SupportedTypes[rtype]
	return ok
}

package zone

import (
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"github.com/hawell/z42/pkg/hiredis"
	"go.uber.org/zap"
	"net/http"
)

type storage interface {
	AddZone(user string, z database.Zone) (int64, error)
	GetZones(user string, start int, count int, q string) ([]string, error)
	GetZone(zone string) (database.Zone, error)
	UpdateZone(z database.Zone) (int64, error)
	DeleteZone(zone string) (int64, error)
	AddLocation(zone string, l database.Location) (int64, error)
	GetLocations(zone string, start int, count int, q string) ([]string, error)
	GetLocation(zone string, location string) (database.Location, error)
	UpdateLocation(zone string, l database.Location) (int64, error)
	DeleteLocation(zone string, location string) (int64, error)
	AddRecordSet(zone string, location string, r database.RecordSet) (int64, error)
	GetRecordSets(zone string, location string) ([]string, error)
	GetRecordSet(zone string, location string, rtype string) (database.RecordSet, error)
	UpdateRecordSet(zone string, location string, r database.RecordSet) (int64, error)
	DeleteRecordSet(zone string, location string, rtype string) (int64, error)
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

func (h *Handler) RegisterHandlers(group *gin.RouterGroup) {
	group.GET("", h.getZones)
	group.POST("", h.addZone)

	group.GET("/:zone", h.getZone)
	group.PUT("/:zone", h.updateZone)
	group.DELETE("/:zone", h.deleteZone)

	group.GET("/:zone/locations", h.getLocations)
	group.POST("/:zone/locations", h.addLocation)

	group.GET("/:zone/locations/:location", h.getLocation)
	group.PUT("/:zone/locations/:location", h.updateLocation)
	group.DELETE("/:zone/locations/:location", h.deleteLocation)

	group.GET("/:zone/locations/:location/rrsets", h.getRecordSets)
	group.POST("/:zone/locations/:location/rrsets", h.addRecordSet)

	group.GET("/:zone/locations/:location/rrsets/:rtype", h.getRecordSet)
	group.PUT("/:zone/locations/:location/rrsets/:rtype", h.updateRecordSet)
	group.DELETE("/:zone/locations/:location/rrsets/:rtype", h.deleteRecordSet)
}

func (h *Handler) getZones(c *gin.Context) {
	user := extractUser(c)
	if user == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	var req listRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	zones, err := h.db.GetZones(user, req.Start, req.Count, req.Q)
	if err != nil {
		zap.L().Error("DataBase.getZones()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	c.JSON(http.StatusOK, zones)
}

func (h *Handler) addZone(c *gin.Context) {
	user := extractUser(c)
	if user == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing")
		return
	}

	var z newZoneRequest
	if err := c.ShouldBindJSON(&z); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.Zone{
		Name:            z.Name,
		Enabled:         z.Enabled,
		Dnssec:          z.Dnssec,
		CNameFlattening: z.CNameFlattening,
	}
	_, err := h.db.AddZone(user, model)
	if err != nil {
		zap.L().Error("DataBase.addZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}

	handlers.SuccessResponse(c, http.StatusCreated, "successful")
}

func (h *Handler) getZone(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	z, err := h.db.GetZone(zone)
	if err != nil {
		zap.L().Error("DataBase.getZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}

	resp := getZoneResponse{
		Name:            z.Name,
		Enabled:         z.Enabled,
		Dnssec:          z.Dnssec,
		CNameFlattening: z.CNameFlattening,
	}

	c.JSON(http.StatusOK, &resp)
}

func (h *Handler) updateZone(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	var req updateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	z := database.Zone{
		Name:            zone,
		Enabled:         req.Enabled,
		Dnssec:          req.Dnssec,
		CNameFlattening: req.CNameFlattening,
	}
	_, err := h.db.UpdateZone(z)
	if err != nil {
		zap.L().Error("DataBase.updateZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusNoContent, "successful")
}

func (h *Handler) deleteZone(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	_, err := h.db.DeleteZone(zone)
	if err != nil {
		zap.L().Error("DataBase.deleteZone()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusNoContent, "successful")
}

func (h *Handler) getLocations(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	var req listRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	locations, err := h.db.GetLocations(zone, req.Start, req.Count, req.Q)
	if err != nil {
		zap.L().Error("DataBase.getLocations()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	c.JSON(http.StatusOK, locations)
}

func (h *Handler) addLocation(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}

	var req newLocationRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.Location{
		Name:    req.Name,
		Enabled: req.Enabled,
	}
	_, err = h.db.AddLocation(zone, model)
	if err != nil {
		zap.L().Error("DataBase.addLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusCreated, "successful")
}

func (h *Handler) getLocation(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	l, err := h.db.GetLocation(zone, location)
	if err != nil {
		zap.L().Error("DataBase.getLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	resp := getLocationResponse{
		Enabled: l.Enabled,
	}
	c.JSON(http.StatusOK, &resp)
}

func (h *Handler) updateLocation(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}

	var req updateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.Location{
		Name:    location,
		Enabled: req.Enabled,
	}
	_, err := h.db.UpdateLocation(zone, model)
	if err != nil {
		zap.L().Error("DataBase.updateLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusNoContent, "successful")
}

func (h *Handler) deleteLocation(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	_, err := h.db.DeleteLocation(zone, location)
	if err != nil {
		zap.L().Error("DataBase.deleteLocation()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusNoContent, "successful")
}

func (h *Handler) getRecordSets(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	rrsets, err := h.db.GetRecordSets(zone, location)
	if err != nil {
		zap.L().Error("DataBase.getRecordSets()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	c.JSON(http.StatusOK, rrsets)
}

func (h *Handler) addRecordSet(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	var req newRecordSetRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.RecordSet{
		Type:    req.Type,
		Value:   req.Value,
		Enabled: req.Enabled,
	}
	_, err = h.db.AddRecordSet(zone, location, model)
	if err != nil {
		zap.L().Error("DataBase.addRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusCreated, "successful")
}

func (h *Handler) getRecordSet(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	rtype := c.Param("rtype")
	if rtype == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "rtype missing")
		return
	}
	r, err := h.db.GetRecordSet(zone, location, rtype)
	if err != nil {
		zap.L().Error("DataBase.getRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	resp := getRecordSetResponse{
		Value:   r.Value,
		Enabled: r.Enabled,
	}
	c.JSON(http.StatusOK, &resp)
}

func (h *Handler) updateRecordSet(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	rtype := c.Param("rtype")
	if rtype == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "rtype missing")
		return
	}
	var req updateRecordSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	model := database.RecordSet{
		Type:    rtype,
		Value:   req.Value,
		Enabled: req.Enabled,
	}
	_, err := h.db.UpdateRecordSet(zone, location, model)
	if err != nil {
		zap.L().Error("DataBase.updateRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusNoContent, "successful")

}

func (h *Handler) deleteRecordSet(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "location missing")
		return
	}
	rtype := c.Param("rtype")
	if rtype == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "rtype missing")
		return
	}
	_, err := h.db.DeleteRecordSet(zone, location, rtype)
	if err != nil {
		zap.L().Error("DataBase.deleteRecordSet()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusNoContent, "successful")
}

func extractUser(c *gin.Context) string {
	user, _ := c.Get(handlers.IdentityKey)
	return user.(*handlers.IdentityData).Email
}

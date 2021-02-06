package server

import (
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

func GetZones(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	user := extractUser(c)
	if user == "" {
		c.String(http.StatusBadRequest, "user missing")
		return
	}

	startStr := c.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startStr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid parameter: start -> %s", startStr)
		return
	}
	countStr := c.DefaultQuery("count", "100")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid parameter: count -> %s", countStr)
		return
	}
	q := c.DefaultQuery("q", "")
	zones, err := db.GetZones(user, start, count, q)
	if err != nil {
		zap.L().Error("DataBase.GetZones()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.JSON(http.StatusOK, zones)
}

func AddZone(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	user := extractUser(c)
	if user == "" {
		c.String(http.StatusBadRequest, "user missing")
		return
	}

	var z database.Zone
	if err := c.ShouldBindJSON(&z); err != nil {
		zap.L().Error("cannot bind form-data to Zone", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid input format")
		return
	}
	_, err := db.AddZone(user, z)
	if err != nil {
		zap.L().Error("DataBase.AddZone()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}

	c.String(http.StatusNoContent, "successful")
}

func GetZone(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}

	z, err := db.GetZone(zone)
	if err != nil {
		zap.L().Error("DataBase.GetZone()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}

	c.JSON(http.StatusOK, &z)
}

func UpdateZone(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}

	var z database.Zone
	if err := c.ShouldBindJSON(&z); err != nil {
		zap.L().Error("cannot bind form-data to Zone", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid input format")
		return
	}
	if z.Name != "" && z.Name != zone {
		c.String(http.StatusBadRequest, "zone mismatch")
		return
	}
	z.Name = zone
	_, err := db.UpdateZone(z)
	if err != nil {
		zap.L().Error("DataBase.UpdateZone()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")
}

func DeleteZone(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	_, err := db.DeleteZone(zone)
	if err != nil {
		zap.L().Error("DataBase.DeleteZone()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")
}

func GetLocations(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}

	startStr := c.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startStr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid parameter: start -> %s", startStr)
		return
	}
	countStr := c.DefaultQuery("count", "100")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid parameter: count -> %s", countStr)
		return
	}
	q := c.DefaultQuery("q", "")
	locations, err := db.GetLocations(zone, start, count, q)
	if err != nil {
		zap.L().Error("DataBase.GetLocations()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.JSON(http.StatusOK, locations)
}

func AddLocation(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}

	var l database.Location
	err := c.ShouldBindJSON(&l)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid input format")
		return
	}
	_, err = db.AddLocation(zone, l)
	if err != nil {
		zap.L().Error("DataBase.AddLocation()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")
}

func GetLocation(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}
	l, err := db.GetLocation(zone, location)
	if err != nil {
		zap.L().Error("DataBase.GetLocation()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.JSON(http.StatusOK, &l)
}

func UpdateLocation(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}

	var l database.Location
	if err := c.ShouldBindJSON(&l); err != nil {
		zap.L().Error("cannot bind form-data to Location", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid input format")
		return
	}
	if l.Name != "" && l.Name != location {
		c.String(http.StatusBadRequest, "location mismatch")
		return
	}
	l.Name = location
	_, err := db.UpdateLocation(zone, l)
	if err != nil {
		zap.L().Error("DataBase.UpdateLocation()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")
}

func DeleteLocation(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}
	_, err := db.DeleteLocation(zone, location)
	if err != nil {
		zap.L().Error("DataBase.DeleteLocation()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")
}

func GetRecordSets(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}
	rrsets, err := db.GetRecordSets(zone, location)
	if err != nil {
		zap.L().Error("DataBase.GetRecordSets()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.JSON(http.StatusOK, rrsets)
}

func AddRecordSet(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}
	var rr database.RecordSet
	err := c.ShouldBindJSON(&rr)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid input format")
		return
	}
	_, err = db.AddRecordSet(zone, location, rr)
	if err != nil {
		zap.L().Error("DataBase.AddRecordSet()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")
}

func GetRecordSet(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}
	rtype := c.Param("rtype")
	if !rtypeValid(rtype) {
		c.String(http.StatusBadRequest, "rtype invalid")
		return
	}
	r, err := db.GetRecordSet(zone, location, rtype)
	if err != nil {
		zap.L().Error("DataBase.GetRecordSet()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.JSON(http.StatusOK, &r)
}

func UpdateRecordSet(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}
	rtype := c.Param("rtype")
	if !rtypeValid(rtype) {
		c.String(http.StatusBadRequest, "invalid rtype")
		return
	}

	var r database.RecordSet
	if err := c.ShouldBindJSON(&r); err != nil {
		zap.L().Error("cannot bind form-data to RecordSet", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid input format")
		return
	}
	if r.Type != "" && r.Type != rtype {
		c.String(http.StatusBadRequest, "type mismatch")
		return
	}
	r.Type = rtype
	_, err := db.UpdateRecordSet(zone, location, r)
	if err != nil {
		zap.L().Error("DataBase.UpdateRecordSet()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")

}

func DeleteRecordSet(c *gin.Context) {
	db := extractDataBase(c)
	if db == nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	zone := c.Param("zone")
	if zone == "" {
		c.String(http.StatusBadRequest, "zone missing")
		return
	}
	location := c.Param("location")
	if location == "" {
		c.String(http.StatusBadRequest, "location missing")
		return
	}
	rtype := c.Param("rtype")
	if !rtypeValid(rtype) {
		c.String(http.StatusBadRequest, "invalid rtype")
		return
	}
	_, err := db.DeleteRecordSet(zone, location, rtype)
	if err != nil {
		zap.L().Error("DataBase.DeleteRecordSet()", zap.Error(err))
		c.String(statusFromError(err))
		return
	}
	c.String(http.StatusNoContent, "successful")
}

func extractDataBase(c *gin.Context) *database.DataBase {
	db, ok := c.MustGet("database").(*database.DataBase)
	if !ok {
		zap.L().Error("no database connection")
		return nil
	}
	return db
}

func extractUser(c *gin.Context) string {
	user, _ := c.Get(identityKey)
	return user.(*database.User).Email
}

func rtypeValid(rtype string) bool {
	if rtype == "" {
		return false
	}
	for _, t := range database.SupportedTypes {
		if rtype == t {
			return true
		}
	}
	return false
}

func statusFromError(err error) (int, string) {
	switch err {
	case database.ErrInvalid:
		return http.StatusForbidden, "invalid request"
	case database.ErrDuplicateEntry:
		return http.StatusConflict, "duplicate entry"
	case database.ErrNotFound:
		return http.StatusNotFound, "entry not found"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}
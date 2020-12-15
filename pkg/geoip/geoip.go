package geoip

import (
	"errors"
	"math"
	"net"

	"github.com/hawell/logger"
	"github.com/oschwald/maxminddb-golang"
)

type GeoIp struct {
	enable    bool
	countryDB *maxminddb.Reader
	asnDB     *maxminddb.Reader
}

type Config struct {
	Enable    bool   `json:"enable"`
	CountryDB string `json:"country_db"`
	ASNDB     string `json:"asn_db"`
}

var (
	ErrGeoIpDisabled = errors.New("geoip is disabled")
	ErrBadDB         = errors.New("bad db")
)

func NewGeoIp(config *Config) *GeoIp {
	g := &GeoIp{
		enable: config.Enable,
	}
	if !g.enable {
		return g
	}
	var err error
	if config.CountryDB != "" {
		g.countryDB, err = maxminddb.Open(config.CountryDB)
		if err != nil {
			logger.Default.Errorf("cannot open maxminddb file %s: %s", config.CountryDB, err)
		}
	}
	if config.ASNDB != "" {
		g.asnDB, err = maxminddb.Open(config.ASNDB)
		if err != nil {
			logger.Default.Errorf("cannot open maxminddb file %s: %s", config.ASNDB, err)
		}
	}

	// defer g.db.Close()
	return g
}

func GetDistance(slat, slong, dlat, dlong float64) float64 {
	deltaLat := (dlat - slat) * math.Pi / 180.0
	deltaLong := (dlong - slong) * math.Pi / 180.0
	slat = slat * math.Pi / 180.0
	dlat = dlat * math.Pi / 180.0

	a := math.Sin(deltaLat/2.0)*math.Sin(deltaLat/2.0) +
		math.Cos(slat)*math.Cos(dlat)*math.Sin(deltaLong/2.0)*math.Sin(deltaLong/2.0)
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))

	// logger.Default.Debugf("distance = %f", c)

	return c
}

func (g *GeoIp) GetCoordinates(ip net.IP) (latitude float64, longitude float64, err error) {
	if !g.enable {
		return 0, 0, ErrGeoIpDisabled
	}
	if g.countryDB == nil {
		return 0, 0, ErrBadDB
	}
	var record struct {
		Location struct {
			Latitude        float64 `maxminddb:"latitude"`
			LongitudeOffset uintptr `maxminddb:"longitude"`
		} `maxminddb:"location"`
	}

	if err := g.countryDB.Lookup(ip, &record); err != nil {
		logger.Default.Errorf("lookup failed : %s", err)
		return 0, 0, err
	}
	_ = g.countryDB.Decode(record.Location.LongitudeOffset, &longitude)
	// logger.Default.Debug("lat = ", record.Location.Latitude, " lang = ", longitude)
	return record.Location.Latitude, longitude, nil
}

func (g *GeoIp) GetCountry(ip net.IP) (string, error) {
	if !g.enable {
		return "", ErrGeoIpDisabled
	}
	if g.countryDB == nil {
		return "", ErrBadDB
	}
	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	// logger.Default.Debugf("ip : %s", ip)
	if err := g.countryDB.Lookup(ip, &record); err != nil {
		logger.Default.Errorf("lookup failed : %s", err)
		return "", err
	}
	// logger.Default.Debug(" country = ", record.Country.ISOCode)
	return record.Country.ISOCode, nil
}

func (g *GeoIp) GetASN(ip net.IP) (uint, error) {
	if !g.enable {
		return 0, ErrGeoIpDisabled
	}
	if g.asnDB == nil {
		return 0, ErrBadDB
	}
	var record struct {
		AutonomousSystemNumber uint `maxminddb:"autonomous_system_number"`
	}
	err := g.asnDB.Lookup(ip, &record)
	if err != nil {
		logger.Default.Errorf("lookup failed : %s", err)
		return 0, err
	}
	// logger.Default.Debug("asn = ", record.AutonomousSystemNumber)
	return record.AutonomousSystemNumber, nil
}

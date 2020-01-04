package handler

import (
	"math"
	"net"

	"github.com/hawell/logger"
	"github.com/oschwald/maxminddb-golang"
)

type GeoIp struct {
	Enable    bool
	CountryDB *maxminddb.Reader
	ASNDB     *maxminddb.Reader
}

type GeoIpConfig struct {
	Enable    bool   `json:"enable"`
	CountryDB string `json:"country_db"`
	ASNDB     string `json:"asn_db"`
}

func NewGeoIp(config *GeoIpConfig) *GeoIp {
	g := &GeoIp{
		Enable: config.Enable,
	}
	var err error
	if g.Enable {
		g.CountryDB, err = maxminddb.Open(config.CountryDB)
		if err != nil {
			logger.Default.Errorf("cannot open maxminddb file %s: %s", config.CountryDB, err)
		}
		g.ASNDB, err = maxminddb.Open(config.ASNDB)
		if err != nil {
			logger.Default.Errorf("cannot open maxminddb file %s: %s", config.ASNDB, err)
		}
	}
	// defer g.db.Close()
	return g
}

func (g *GeoIp) GetSameCountry(sourceIp net.IP, ips []IP_RR, mask []int) []int {
	if !g.Enable || g.CountryDB == nil {
		return mask
	}
	sourceCountry, err := g.GetCountry(sourceIp)
	if err != nil {
		logger.Default.Error("getSameCountry failed")
		return mask
	}

	passed := 0
	if sourceCountry != "" {
		outer:
		for i, x := range mask {
			if x == IpMaskWhite {
				for _, country := range ips[i].Country {
					if country == sourceCountry {
						passed++
						continue outer
					}
				}
				mask[i] = IpMaskGrey
			} else {
				mask[i] = IpMaskBlack
			}
		}
	}
	if passed > 0 {
		return mask
	}

	for i, x := range mask {
		if x != IpMaskBlack {
			if ips[i].Country == nil || len(ips[i].Country) == 0 {
				mask[i] = IpMaskWhite
			} else {
				mask[i] = IpMaskBlack
				for _, country := range ips[i].Country {
					if country == "" {
						mask[i] = IpMaskWhite
						break
					}
				}
			}
		}
	}

	return mask
}

func (g *GeoIp) GetSameASN(sourceIp net.IP, ips []IP_RR, mask []int) []int {
	if !g.Enable || g.ASNDB == nil {
		return mask
	}
	sourceASN, err := g.GetASN(sourceIp)
	if err != nil {
		logger.Default.Error("getSameASN failed")
		return mask
	}

	passed := 0
	if sourceASN != 0 {
		outer:
		for i, x := range mask {
			if x == IpMaskWhite {
				for _, asn := range ips[i].ASN {
					if asn == sourceASN {
						passed++
						continue outer
					}
				}
				mask[i] = IpMaskGrey
			} else {
				mask[i] = IpMaskBlack
			}
		}
	}
	if passed > 0 {
		return mask
	}

	for i, x := range mask {
		if x != IpMaskBlack {
			if ips[i].ASN == nil || len(ips[i].ASN) == 0 {
				mask[i] = IpMaskWhite
			} else {
				mask[i] = IpMaskBlack
				for _, asn := range ips[i].ASN {
					if asn == 0 {
						mask[i] = IpMaskWhite
						break
					}
				}
			}
		}
	}

	return mask
}

// TODO: add a margin for minimum distance
func (g *GeoIp) GetMinimumDistance(sourceIp net.IP, ips []IP_RR, mask []int) []int {
	if !g.Enable || g.CountryDB == nil {
		return mask
	}
	minDistance := 1000.0
	dists := make([]float64, 0, len(mask))
	slat, slong, err := g.GetCoordinates(sourceIp)
	if err != nil {
		logger.Default.Error("getMinimumDistance failed")
		return mask
	}
	for i, x := range mask {
		if x == IpMaskWhite {
			destinationIp := ips[i].Ip
			dlat, dlong, _ := g.GetCoordinates(destinationIp)
			d, err := g.getDistance(slat, slong, dlat, dlong)
			if err != nil {
				d = 1000.0
			}
			if d < minDistance {
				minDistance = d
			}
			dists = append(dists, d)
		}
	}

	passed := 0
	for i, x := range mask {
		if x == IpMaskWhite {
			if dists[i] == minDistance {
				passed++
			} else {
				mask[i] = IpMaskGrey
			}
		} else {
			mask[i] = IpMaskBlack
		}
	}
	if passed > 0 {
		return mask
	} else {
		for i := range mask {
			if mask[i] == IpMaskGrey {
				mask[i] = IpMaskWhite
			}
		}
		return mask
	}
}

func (g *GeoIp) getDistance(slat, slong, dlat, dlong float64) (float64, error) {
	deltaLat := (dlat - slat) * math.Pi / 180.0
	deltaLong := (dlong - slong) * math.Pi / 180.0
	slat = slat * math.Pi / 180.0
	dlat = dlat * math.Pi / 180.0

	a := math.Sin(deltaLat/2.0)*math.Sin(deltaLat/2.0) +
		math.Cos(slat)*math.Cos(dlat)*math.Sin(deltaLong/2.0)*math.Sin(deltaLong/2.0)
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))

	// logger.Default.Debugf("distance = %f", c)

	return c, nil
}

func (g *GeoIp) GetCoordinates(ip net.IP) (latitude float64, longitude float64, err error) {
	if !g.Enable || g.CountryDB == nil {
		return
	}
	var record struct {
		Location struct {
			Latitude        float64 `maxminddb:"latitude"`
			LongitudeOffset uintptr `maxminddb:"longitude"`
		} `maxminddb:"location"`
	}

	if err := g.CountryDB.Lookup(ip, &record); err != nil {
		logger.Default.Errorf("lookup failed : %s", err)
		return 0, 0, err
	}
	_ = g.CountryDB.Decode(record.Location.LongitudeOffset, &longitude)
	// logger.Default.Debug("lat = ", record.Location.Latitude, " lang = ", longitude)
	return record.Location.Latitude, longitude, nil
}

func (g *GeoIp) GetCountry(ip net.IP) (country string, err error) {
	if !g.Enable || g.CountryDB == nil {
		return
	}
	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	// logger.Default.Debugf("ip : %s", ip)
	if err := g.CountryDB.Lookup(ip, &record); err != nil {
		logger.Default.Errorf("lookup failed : %s", err)
		return "", err
	}
	// logger.Default.Debug(" country = ", record.Country.ISOCode)
	return record.Country.ISOCode, nil
}

func (g *GeoIp) GetASN(ip net.IP) (uint, error) {
	var record struct {
		AutonomousSystemNumber uint `maxminddb:"autonomous_system_number"`
	}
	err := g.ASNDB.Lookup(ip, &record)
	if err != nil {
		logger.Default.Errorf("lookup failed : %s", err)
		return 0, err
	}
	// logger.Default.Debug("asn = ", record.AutonomousSystemNumber)
	return record.AutonomousSystemNumber, nil
}

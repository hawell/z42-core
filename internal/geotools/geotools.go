package geotools

import (
	"z42-core/internal/types"
	"z42-core/pkg/geoip"
	"go.uber.org/zap"
	"net"
)

func GetSameCountry(g *geoip.GeoIp, sourceIp net.IP, ips []types.IP_RR, mask []int) ([]int, error) {
	sourceCountry, err := g.GetCountry(sourceIp)
	if err != nil {
		zap.L().Error("getSameCountry failed")
		return mask, err
	}

	passed := 0
	if sourceCountry != "" {
	outer:
		for i, x := range mask {
			if x == types.IpMaskWhite {
				for _, country := range ips[i].Country {
					if country == sourceCountry {
						passed++
						continue outer
					}
				}
				mask[i] = types.IpMaskGrey
			} else {
				mask[i] = types.IpMaskBlack
			}
		}
	}
	if passed > 0 {
		return mask, nil
	}

	for i, x := range mask {
		if x != types.IpMaskBlack {
			if ips[i].Country == nil || len(ips[i].Country) == 0 {
				mask[i] = types.IpMaskWhite
			} else {
				mask[i] = types.IpMaskBlack
				for _, country := range ips[i].Country {
					if country == "" {
						mask[i] = types.IpMaskWhite
						break
					}
				}
			}
		}
	}

	return mask, nil
}

func GetSameASN(g *geoip.GeoIp, sourceIp net.IP, ips []types.IP_RR, mask []int) ([]int, error) {
	sourceASN, err := g.GetASN(sourceIp)
	if err != nil {
		zap.L().Error("getSameASN failed")
		return mask, err
	}

	passed := 0
	if sourceASN != 0 {
	outer:
		for i, x := range mask {
			if x == types.IpMaskWhite {
				for _, asn := range ips[i].ASN {
					if asn == sourceASN {
						passed++
						continue outer
					}
				}
				mask[i] = types.IpMaskGrey
			} else {
				mask[i] = types.IpMaskBlack
			}
		}
	}
	if passed > 0 {
		return mask, nil
	}

	for i, x := range mask {
		if x != types.IpMaskBlack {
			if ips[i].ASN == nil || len(ips[i].ASN) == 0 {
				mask[i] = types.IpMaskWhite
			} else {
				mask[i] = types.IpMaskBlack
				for _, asn := range ips[i].ASN {
					if asn == 0 {
						mask[i] = types.IpMaskWhite
						break
					}
				}
			}
		}
	}

	return mask, nil
}

// TODO: add a margin for minimum distance
func GetMinimumDistance(g *geoip.GeoIp, sourceIp net.IP, ips []types.IP_RR, mask []int) ([]int, error) {
	minDistance := 1000.0
	dists := make([]float64, 0, len(mask))
	slat, slong, err := g.GetCoordinates(sourceIp)
	if err != nil {
		zap.L().Error("getMinimumDistance failed")
		return mask, err
	}
	for i, x := range mask {
		if x == types.IpMaskWhite {
			destinationIp := ips[i].Ip
			dlat, dlong, _ := g.GetCoordinates(destinationIp)
			d := geoip.GetDistance(slat, slong, dlat, dlong)
			if d < minDistance {
				minDistance = d
			}
			dists = append(dists, d)
		}
	}

	passed := 0
	for i, x := range mask {
		if x == types.IpMaskWhite {
			if dists[i] == minDistance {
				passed++
			} else {
				mask[i] = types.IpMaskGrey
			}
		} else {
			mask[i] = types.IpMaskBlack
		}
	}
	if passed > 0 {
		return mask, nil
	} else {
		for i := range mask {
			if mask[i] == types.IpMaskGrey {
				mask[i] = types.IpMaskWhite
			}
		}
		return mask, nil
	}
}

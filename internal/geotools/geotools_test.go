package geotools

import (
	"z42-core/internal/types"
	"z42-core/pkg/geoip"
	. "github.com/onsi/gomega"
	"log"
	"net"
	"strconv"
	"testing"
)

var (
	asnDB     = "../../assets/geoIsp.mmdb"
	countryDB = "../../assets/geoCity.mmdb"
)

func TestGeoIpAutomatic(t *testing.T) {
	sip := [][]string{
		{"212.83.32.45", "DE", "213.95.10.76"},
		{"80.67.163.250", "FR", "62.240.228.4"},
		{"178.18.89.144", "NL", "46.19.36.12"},
		{"206.108.0.43", "CA", "154.11.253.242"},
		{"185.70.144.117", "DE", "213.95.10.76"},
		{"62.220.128.73", "CH", "82.220.3.51"},
	}

	dip := [][]string{
		{"82.220.3.51", "CH"},
		{"192.30.252.225", "US"},
		{"213.95.10.76", "DE"},
		{"94.76.229.204", "GB"},
		{"46.19.36.12", "NL"},
		{"46.30.209.1", "DK"},
		{"91.239.97.26", "SI"},
		{"14.1.44.230", "NZ"},
		{"52.76.214.87", "SG"},
		{"103.31.84.12", "MV"},
		{"212.63.210.241", "SE"},
		{"154.11.253.242", "CA"},
		{"128.139.197.81", "IL"},
		{"194.190.198.13", "RU"},
		{"84.88.14.229", "ES"},
		{"79.110.197.36", "PL"},
		{"175.45.73.66", "AU"},
		{"62.240.228.4", "FR"},
		{"200.238.130.54", "BR"},
		{"13.113.70.195", "JP"},
		{"37.252.235.214", "AT"},
		{"185.87.111.13", "FI"},
		{"52.66.51.117", "IN"},
		{"193.198.233.217", "HR"},
		{"118.67.200.190", "KH"},
		{"103.6.84.107", "HK"},
		{"78.128.211.50", "CZ"},
		{"87.238.39.42", "NO"},
		{"37.148.176.54", "BE"},
	}

	RegisterTestingT(t)
	cfg := geoip.Config{
		Enable:    true,
		CountryDB: countryDB,
	}

	geoIp := geoip.NewGeoIp(&cfg)

	for i := range sip {
		dest := new(types.IP_RRSet)
		for j := range dip {
			cc, _ := geoIp.GetCountry(net.ParseIP(dip[j][0]))
			Expect(cc).To(Equal(dip[j][1]))
			r := types.IP_RR{
				Ip: net.ParseIP(dip[j][0]),
			}
			dest.Data = append(dest.Data, r)
		}
		dest.TtlValue = 100
		mask := make([]int, len(dest.Data))
		mask, err := GetMinimumDistance(geoIp, net.ParseIP(sip[i][0]), dest.Data, mask)
		Expect(err).To(BeNil())
		index := 0
		for j, x := range mask {
			if x == types.IpMaskWhite {
				index = j
				break
			}
		}
		log.Println("[DEBUG]", sip[i][0], " ", dest.Data[index].Ip.String())
		Expect(sip[i][2]).To(Equal(dest.Data[index].Ip.String()))
	}
}

func TestGetSameCountry(t *testing.T) {
	sip := [][]string{
		{"212.83.32.45", "DE", "1.2.3.4"},
		{"80.67.163.250", "FR", "2.3.4.5"},
		{"154.11.253.242", "", "3.4.5.6"},
		{"127.0.0.1", "", "3.4.5.6"},
	}

	RegisterTestingT(t)
	cfg := geoip.Config{
		Enable:    true,
		CountryDB: countryDB,
	}

	geoIp := geoip.NewGeoIp(&cfg)

	for i := range sip {
		var dest types.IP_RRSet
		dest.Data = []types.IP_RR{
			{Ip: net.ParseIP("1.2.3.4"), Country: []string{"DE"}},
			{Ip: net.ParseIP("2.3.4.5"), Country: []string{"FR"}},
			{Ip: net.ParseIP("3.4.5.6"), Country: []string{""}},
		}
		mask := make([]int, len(dest.Data))
		mask, err := GetSameCountry(geoIp, net.ParseIP(sip[i][0]), dest.Data, mask)
		Expect(err).To(BeNil())
		index := -1
		for j, x := range mask {
			if x == types.IpMaskWhite {
				index = j
				break
			}
		}
		Expect(index).NotTo(Equal(-1))
		log.Println("[DEBUG]", sip[i][1], sip[i][2], dest.Data[index].Country, dest.Data[index].Ip.String())
		Expect(dest.Data[index].Country[0]).To(Equal(sip[i][1]))
		Expect(dest.Data[index].Ip.String()).To(Equal(sip[i][2]))
	}

}

func TestGetSameASN(t *testing.T) {
	sip := []string{
		"212.83.32.45",
		"80.67.163.250",
		"154.11.253.242",
		"127.0.0.1",
	}

	dip := types.IP_RRSet{
		Data: []types.IP_RR{
			{Ip: net.ParseIP("1.2.3.4"), ASN: []uint{47447}},
			{Ip: net.ParseIP("2.3.4.5"), ASN: []uint{20766}},
			{Ip: net.ParseIP("3.4.5.6"), ASN: []uint{852}},
			{Ip: net.ParseIP("4.5.6.7"), ASN: []uint{0}},
		},
	}

	res := [][]string{
		{"47447", "1.2.3.4"},
		{"20766", "2.3.4.5"},
		{"852", "3.4.5.6"},
		{"0", "4.5.6.7"},
	}
	cfg := geoip.Config{
		Enable: true,
		ASNDB:  asnDB,
	}
	RegisterTestingT(t)

	geoIp := geoip.NewGeoIp(&cfg)

	for i := range sip {
		mask := make([]int, len(dip.Data))
		mask, err := GetSameASN(geoIp, net.ParseIP(sip[i]), dip.Data, mask)
		Expect(err).To(BeNil())
		index := -1
		for j, x := range mask {
			if x == types.IpMaskWhite {
				index = j
				break
			}
		}
		Expect(index).NotTo(Equal(-1))
		Expect(strconv.Itoa(int(dip.Data[index].ASN[0]))).To(Equal(res[i][0]))
		Expect(dip.Data[index].Ip.String()).To(Equal(res[i][1]))
	}
}

func TestDisabled(t *testing.T) {
	cfg := geoip.Config{
		Enable:    false,
		CountryDB: countryDB,
		ASNDB:     asnDB,
	}
	RegisterTestingT(t)

	geoIp := geoip.NewGeoIp(&cfg)

	_, err := GetMinimumDistance(geoIp, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	Expect(err).To(Equal(geoip.ErrGeoIpDisabled))
	_, err = GetSameASN(geoIp, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	Expect(err).To(Equal(geoip.ErrGeoIpDisabled))
	_, err = GetSameCountry(geoIp, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	Expect(err).To(Equal(geoip.ErrGeoIpDisabled))
}

func TestBadDB(t *testing.T) {
	cfg := geoip.Config{
		Enable:    true,
		CountryDB: "ddd",
		ASNDB:     "ddds",
	}
	RegisterTestingT(t)
	geoIp := geoip.NewGeoIp(&cfg)

	_, err := GetMinimumDistance(geoIp, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	Expect(err).To(Equal(geoip.ErrBadDB))
	_, err = GetSameASN(geoIp, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	Expect(err).To(Equal(geoip.ErrBadDB))
	_, err = GetSameCountry(geoIp, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	Expect(err).To(Equal(geoip.ErrBadDB))
}

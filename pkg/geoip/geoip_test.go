package geoip

import (
	"fmt"
	. "github.com/onsi/gomega"
	"net"
	"testing"
)

var (
	asnDB     = "../../assets/geoIsp.mmdb"
	countryDB = "../../assets/geoCity.mmdb"
)

/*
82.220.3.51 9044 CH
192.30.252.225 36459 US
213.95.10.76 12337 DE
94.76.229.204 29550 GB
46.19.36.12 196752 NL
46.30.209.1 51468 DK
91.239.97.26 198644 SI
14.1.44.230 45177 NZ
52.76.214.87 16509 SG
103.31.84.12 7642 MV
212.63.210.241 30880 SE
154.11.253.242 852 CA
128.139.197.81 378 IL
194.190.198.13 2118 RU
84.88.14.229 13041 ES
79.110.197.36 35179 PL
175.45.73.66 4826 AU
62.240.228.4 8426 FR
200.238.130.54 10881 BR
13.113.70.195 16509 JP
37.252.235.214 42473 AT
185.87.111.13 201057 FI
52.66.51.117 16509 IN
193.198.233.217 2108 HR
118.67.200.190 7712 KH
103.6.84.107 36236 HK
78.128.211.50 2852 CZ
87.238.39.42 39029 NO
37.148.176.54 34762 BE
212.83.32.45 47447 DE
80.67.163.250 20766 FR
178.18.89.144 35470 NL
206.108.0.43 393424 CA
185.70.144.117 200567 DE
62.220.128.73 6893 CH
*/
func _() {
	ips := []string{
		"82.220.3.51",
		"192.30.252.225",
		"213.95.10.76",
		"94.76.229.204",
		"46.19.36.12",
		"46.30.209.1",
		"91.239.97.26",
		"14.1.44.230",
		"52.76.214.87",
		"103.31.84.12",
		"212.63.210.241",
		"154.11.253.242",
		"128.139.197.81",
		"194.190.198.13",
		"84.88.14.229",
		"79.110.197.36",
		"175.45.73.66",
		"62.240.228.4",
		"200.238.130.54",
		"13.113.70.195",
		"37.252.235.214",
		"185.87.111.13",
		"52.66.51.117",
		"193.198.233.217",
		"118.67.200.190",
		"103.6.84.107",
		"78.128.211.50",
		"87.238.39.42",
		"37.148.176.54",
		"212.83.32.45",
		"80.67.163.250",
		"178.18.89.144",
		"206.108.0.43",
		"185.70.144.117",
		"62.220.128.73",
	}
	cfg := Config{
		Enable:    true,
		ASNDB:     asnDB,
		CountryDB: countryDB,
	}

	g := NewGeoIp(&cfg)

	for _, ip := range ips {
		asn, _ := g.GetASN(net.ParseIP(ip))
		c, _ := g.GetCountry(net.ParseIP(ip))
		fmt.Println(ip, asn, c)
	}
}

func TestDisabled(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Enable:    false,
		CountryDB: countryDB,
		ASNDB:     asnDB,
	}
	geoIp := NewGeoIp(&cfg)

	Expect(geoIp.enable).To(BeFalse())

	_, err := geoIp.GetASN(net.ParseIP("1.2.3.4"))
	Expect(err).To(Equal(ErrGeoIpDisabled))
	_, err = geoIp.GetCountry(net.ParseIP("1.2.3.4"))
	Expect(err).To(Equal(ErrGeoIpDisabled))
	_, _, err = geoIp.GetCoordinates(net.ParseIP("1.2.3.4"))
	Expect(err).To(Equal(ErrGeoIpDisabled))
}

func TestBadDB(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Enable:    true,
		CountryDB: "ddd",
		ASNDB:     "ddds",
	}
	geoIp := NewGeoIp(&cfg)

	Expect(geoIp.enable).To(BeTrue())

	_, err := geoIp.GetASN(net.ParseIP("1.2.3.4"))
	Expect(err).To(Equal(ErrBadDB))
	_, err = geoIp.GetCountry(net.ParseIP("1.2.3.4"))
	Expect(err).To(Equal(ErrBadDB))
	_, _, err = geoIp.GetCoordinates(net.ParseIP("1.2.3.4"))
	Expect(err).To(Equal(ErrBadDB))
}

package geoip

import (
	"fmt"
	"z42-core/configs"
	"github.com/oschwald/maxminddb-golang"
	"net"
	"os"
)

type Config struct {
	Enable    bool   `json:"enable"`
	CountryDB string `json:"country_db"`
	ASNDB     string `json:"asn_db"`
}

func DefaultConfig() Config {
	return Config{
		Enable:    false,
		CountryDB: "geoCity.mmdb",
		ASNDB:     "geoIsp.mmdb",
	}
}

func (c Config) Verify() {
	if c.Enable {
		fmt.Println("checking geoip...")
		var countryRecord struct {
			Location struct {
				Latitude        float64 `maxminddb:"latitude"`
				LongitudeOffset uintptr `maxminddb:"longitude"`
			} `maxminddb:"location"`
			Country struct {
				ISOCode string `maxminddb:"iso_code"`
			} `maxminddb:"country"`
		}
		var asnRecord struct {
			AutonomousSystemNumber uint `maxminddb:"autonomous_system_number"`
		}
		records := []interface{}{countryRecord, asnRecord}
		var err error
		for i, dbFile := range []string{c.CountryDB, c.ASNDB} {
			msg := fmt.Sprintf("checking file stat : %s", dbFile)
			_, err = os.Stat(dbFile)
			configs.PrintResult(msg, err)
			if err == nil {
				msg = fmt.Sprintf("checking db : %s", dbFile)
				var db *maxminddb.Reader
				db, err = maxminddb.Open(dbFile)
				configs.PrintResult(msg, err)
				if err == nil {
					msg = fmt.Sprintf("checking db query results")
					err = db.Lookup(net.ParseIP("46.19.36.12"), &records[i])
					configs.PrintResult(msg, err)
				}
			}
		}
	}
}

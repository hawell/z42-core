package healthcheck

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"log"
	"net"
	"strconv"
	"testing"
	"time"
	"z42-core/internal/storage"
	"z42-core/internal/types"
	"z42-core/pkg/hiredis"
)

var healthcheckGetEntries = []*types.HealthCheckItem{
	{Host: "w0.healthcheck.com.", Ip: "1.2.3.4", Protocol: "http", Uri: "/", Port: 80, Status: 3, Enable: true},
	{Host: "w0.healthcheck.com.", Ip: "2.3.4.5", Protocol: "http", Uri: "/", Port: 80, Status: 1, Enable: true},
	{Host: "w0.healthcheck.com.", Ip: "3.4.5.6", Protocol: "http", Uri: "/", Port: 80, Status: 0, Enable: true},
	{Host: "w0.healthcheck.com.", Ip: "4.5.6.7", Protocol: "http", Uri: "/", Port: 80, Status: -1, Enable: true},
	{Host: "w0.healthcheck.com.", Ip: "5.6.7.8", Protocol: "http", Uri: "/", Port: 80, Status: -3, Enable: true},

	{Host: "w1.healthcheck.com.", Ip: "2.3.4.5", Protocol: "http", Uri: "/", Port: 80, Status: 1, Enable: true},
	{Host: "w1.healthcheck.com.", Ip: "3.4.5.6", Protocol: "http", Uri: "/", Port: 80, Status: 0, Enable: true},
	{Host: "w1.healthcheck.com.", Ip: "4.5.6.7", Protocol: "http", Uri: "/", Port: 80, Status: -1, Enable: true},
	{Host: "w1.healthcheck.com.", Ip: "5.6.7.8", Protocol: "http", Uri: "/", Port: 80, Status: -3, Enable: true},

	{Host: "w2.healthcheck.com.", Ip: "3.4.5.6", Protocol: "http", Uri: "/", Port: 80, Status: 0, Enable: true},
	{Host: "w2.healthcheck.com.", Ip: "4.5.6.7", Protocol: "http", Uri: "/", Port: 80, Status: -1, Enable: true},
	{Host: "w2.healthcheck.com.", Ip: "5.6.7.8", Protocol: "http", Uri: "/", Port: 80, Status: -3, Enable: true},

	{Host: "w3.healthcheck.com.", Ip: "4.5.6.7", Protocol: "http", Uri: "/", Port: 80, Status: -1, Enable: true},
	{Host: "w3.healthcheck.com.", Ip: "5.6.7.8", Protocol: "http", Uri: "/", Port: 80, Status: -3, Enable: true},

	{Host: "w4.healthcheck.com.", Ip: "5.6.7.8", Protocol: "http", Uri: "/", Port: 80, Status: -3, Enable: true},
}

var stats = []int{3, 1, 0, -1, -3, 1, 0, -1, -3, 0, -1, -3, -1, -3, -3}
var filterResult = []int{1, 3, 2, 1, 1}

var healthCheckSetEntries = [][]string{
	{"@", "185.143.233.2",
		`{"host": "healthcheck.com.", "ip": "185.143.233.2", enable":true,"protocol":"http","uri":"","port":80, "timeout": 1000}`,
	},
	{"www", "185.143.234.50",
		`{"host": "www.healthcheck.com.", "ip": "185.143.233.2", "enable":true,"protocol":"http","uri":"","port":80, "timeout": 1000}`,
	},
}

var healthcheckRedisStatConfig = storage.StatHandlerConfig{
	Redis: hiredis.Config{
		Address:  "127.0.0.1:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "healthcheck_",
		Suffix:   "_healthcheck",
		Connection: hiredis.ConnectionConfig{
			MaxIdleConnections:   10,
			MaxActiveConnections: 10,
			ConnectTimeout:       500,
			ReadTimeout:          500,
			IdleKeepAlive:        30,
			MaxKeepAlive:         0,
			WaitForConnection:    true,
		},
	},
}

var healthcheckRedisDataConfig = storage.DataHandlerConfig{
	ZoneCacheSize:      10000,
	ZoneCacheTimeout:   60,
	ZoneReload:         60,
	RecordCacheSize:    10000000,
	RecordCacheTimeout: 1,
	MinTTL:             5,
	MaxTTL:             300,
	Redis: hiredis.Config{
		Address:  "127.0.0.1:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "hcconfig_",
		Suffix:   "_hcconfig",
		Connection: hiredis.ConnectionConfig{
			MaxIdleConnections:   10,
			MaxActiveConnections: 10,
			ConnectTimeout:       500,
			ReadTimeout:          500,
			IdleKeepAlive:        30,
			MaxKeepAlive:         0,
			WaitForConnection:    true,
		},
	},
}

var healthcheckTestConfig = Config{
	Enable:             true,
	MaxRequests:        10,
	MaxPendingRequests: 100,
	UpdateInterval:     600,
	CheckInterval:      600,
}

func TestGet(t *testing.T) {
	RegisterTestingT(t)
	log.Println("TestGet")
	dh := storage.NewDataHandler(&healthcheckRedisDataConfig)
	dh.Start()
	sh := storage.NewStatHandler(&healthcheckRedisStatConfig)
	l, _ := zap.NewProduction()
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh, l)

	h.redisStat.Clear()
	h.redisData.Clear()
	for _, entry := range healthcheckGetEntries {
		h.redisStat.SetHealthcheckItem(entry)
	}

	for i, item := range healthcheckGetEntries {
		stat := h.redisStat.GetHealthStatus(item.Host, item.Ip)
		log.Println("[DEBUG]", stat, " ", stats[i])
		Expect(stat).To(Equal(stats[i]))
	}
	// h.Stop()
	h.redisStat.Clear()
}

func TestFilter(t *testing.T) {
	RegisterTestingT(t)
	log.Println("TestFilter")
	dh := storage.NewDataHandler(&healthcheckRedisDataConfig)
	dh.Start()
	sh := storage.NewStatHandler(&healthcheckRedisStatConfig)
	l, _ := zap.NewProduction()
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh, l)

	h.redisStat.Clear()
	h.redisData.Clear()
	for _, entry := range healthcheckGetEntries {
		h.redisStat.SetHealthcheckItem(entry)
	}

	w := []types.IP_RRSet{
		{
			Data: []types.IP_RR{
				{Ip: net.ParseIP("1.2.3.4")},
				{Ip: net.ParseIP("2.3.4.5")},
				{Ip: net.ParseIP("3.4.5.6")},
				{Ip: net.ParseIP("4.5.6.7")},
				{Ip: net.ParseIP("5.6.7.8")},
			},
			FilterConfig: types.IpFilterConfig{
				Count:     "multi",
				Order:     "none",
				GeoFilter: "none",
			},
			HealthCheckConfig: types.IpHealthCheckConfig{
				Enable:    true,
				DownCount: -3,
				UpCount:   3,
				Timeout:   1000,
			},
		},
		{
			Data: []types.IP_RR{
				{Ip: net.ParseIP("2.3.4.5")},
				{Ip: net.ParseIP("3.4.5.6")},
				{Ip: net.ParseIP("4.5.6.7")},
				{Ip: net.ParseIP("5.6.7.8")},
			},
			FilterConfig: types.IpFilterConfig{
				Count:     "multi",
				Order:     "none",
				GeoFilter: "none",
			},
			HealthCheckConfig: types.IpHealthCheckConfig{
				Enable:    true,
				DownCount: -3,
				UpCount:   3,
				Timeout:   1000,
			},
		},
		{
			Data: []types.IP_RR{
				{Ip: net.ParseIP("3.4.5.6")},
				{Ip: net.ParseIP("4.5.6.7")},
				{Ip: net.ParseIP("5.6.7.8")},
			},
			FilterConfig: types.IpFilterConfig{
				Count:     "multi",
				Order:     "none",
				GeoFilter: "none",
			},
			HealthCheckConfig: types.IpHealthCheckConfig{
				Enable:    true,
				DownCount: -3,
				UpCount:   3,
				Timeout:   1000,
			},
		},
		{
			Data: []types.IP_RR{
				{Ip: net.ParseIP("4.5.6.7")},
				{Ip: net.ParseIP("5.6.7.8")},
			},
			FilterConfig: types.IpFilterConfig{
				Count:     "multi",
				Order:     "none",
				GeoFilter: "none",
			},
			HealthCheckConfig: types.IpHealthCheckConfig{
				Enable:    true,
				DownCount: -3,
				UpCount:   3,
				Timeout:   1000,
			},
		},
		{
			Data: []types.IP_RR{
				{Ip: net.ParseIP("5.6.7.8")},
			},
			FilterConfig: types.IpFilterConfig{
				Count:     "multi",
				Order:     "none",
				GeoFilter: "none",
			},
			HealthCheckConfig: types.IpHealthCheckConfig{
				Enable:    true,
				DownCount: -3,
				UpCount:   3,
				Timeout:   1000,
			},
		},
	}
	for i := range w {
		log.Println("[DEBUG]", w[i])
		mask := make([]int, len(w[i].Data))
		mask = h.FilterHealthcheck("w"+strconv.Itoa(i)+".healthcheck.com.", &w[i], mask)
		log.Println("[DEBUG]", w[i])
		count := 0
		for _, x := range mask {
			if x == types.IpMaskWhite {
				count++
			}
		}
		Expect(count).To(Equal(filterResult[i]))
	}
	h.redisStat.Clear()
	// h.Stop()
}

func TestSet(t *testing.T) {
	log.Println("TestSet")
	dh := storage.NewDataHandler(&healthcheckRedisDataConfig)
	dh.Start()
	sh := storage.NewStatHandler(&healthcheckRedisStatConfig)
	l, _ := zap.NewProduction()
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh, l)

	h.redisStat.Clear()
	h.redisData.Clear()
	for _, str := range healthCheckSetEntries {
		a := fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", str[1], str[2])
		h.redisData.SetLocationFromJson("healthcheck.com.", str[0], a)
		var item types.HealthCheckItem
		jsoniter.Unmarshal([]byte(str[2]), &item)
		h.redisStat.SetHealthcheckItem(&item)
	}
	// h.transferItems()
	go h.Start()
	time.Sleep(time.Second * 10)

	log.Println("[DEBUG]", h.redisStat.GetHealthStatus("example.com", "185.143.233.2"))
	log.Println("[DEBUG]", h.redisStat.GetHealthStatus("www.example.com", "185.143.234.50"))
}

func TestTransfer(t *testing.T) {
	RegisterTestingT(t)
	log.Printf("TestTransfer")

	var healthcheckTransferItems = [][]string{
		{"w0", "1.2.3.4",
			`{"host":"w0.healthcheck.com.", "ip":"1.2.3.4", "enable":true, "protocol":"http", "uri":"/uri0", "port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
			`{"host":"w0.healthcheck.com.", "ip":"1.2.3.4", "enable":true, "protocol":"http", "uri":"/uri0", "port":80, "status":2, "up_count": 3, "down_count": -3, "timeout":1000}`,
		},
		{"w1", "2.3.4.5",
			`{"host":"w1.healthcheck.com.", "ip":"2.3.4.5", "enable":true, "protocol":"https", "uri":"/uri111", "port":8081, "up_count": 3, "down_count": -3, "timeout":1000}`,
			`{"host":"w1.healthcheck.com.", "ip":"2.3.4.5", "enable":true, "protocol":"http", "uri":"/uri1", "port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
		},
		{"w2", "3.4.5.6",
			"",
			`{"host":"w2.healthcheck.com.", "ip":"3.4.5.6", "enable":true, "protocol":"http", "uri":"/uri2", "port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
		},
		{"w3", "4.5.6.7",
			`{"host":"w3.healthcheck.com.", "ip":"4.5.6.7", "enable":true, "protocol":"http", "uri":"/uri3", "port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
			``,
		},
	}

	var healthCheckTransferResults = []*types.HealthCheckItem{
		{Host: "w0.healthcheck.com.", Ip: "1.2.3.4", Protocol: "http", Uri: "/uri0", Port: 80, Status: 2, Timeout: 1000, UpCount: 3, DownCount: -3, Enable: true},
		{Host: "w1.healthcheck.com.", Ip: "2.3.4.5", Protocol: "https", Uri: "/uri111", Port: 8081, Status: 0, Timeout: 1000, UpCount: 3, DownCount: -3, Enable: true},
		{Host: "w3.healthcheck.com.", Ip: "4.5.6.7", Protocol: "http", Uri: "/uri3", Port: 80, Status: 0, Timeout: 1000, UpCount: 3, DownCount: -3, Enable: true},
	}

	dh := storage.NewDataHandler(&healthcheckRedisDataConfig)
	dh.Start()
	sh := storage.NewStatHandler(&healthcheckRedisStatConfig)
	l, _ := zap.NewProduction()
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh, l)

	h.redisData.Clear()
	h.redisStat.Clear()
	h.redisData.EnableZone("healthcheck.com.")
	for _, str := range healthcheckTransferItems {
		if str[2] != "" {
			a := fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", str[1], str[2])
			h.redisData.SetLocationFromJson("healthcheck.com.", str[0], a)
		}
		if str[3] != "" {
			var item types.HealthCheckItem
			jsoniter.Unmarshal([]byte(str[3]), &item)
			h.redisStat.SetHealthcheckItem(&item)
		}
	}

	// h.transferItems()
	go h.Start()
	time.Sleep(time.Second * 10)

	itemsEqual := func(item1 *types.HealthCheckItem, item2 *types.HealthCheckItem) bool {
		if item1.Ip != item2.Ip || item1.Uri != item2.Uri || item1.Port != item2.Port ||
			item1.Protocol != item2.Protocol || item1.Enable != item2.Enable ||
			item1.UpCount != item2.UpCount || item1.DownCount != item2.DownCount || item1.Timeout != item2.Timeout {
			return false
		}
		return true
	}

	for _, item := range healthCheckTransferResults {
		key := item.Host + ":" + item.Ip
		storedItem, _ := h.redisStat.GetHealthcheckItem(key)
		Expect(itemsEqual(item, storedItem)).To(BeTrue())
	}
}

func TestHealthCheck(t *testing.T) {
	RegisterTestingT(t)
	var healthcheckStatConfig = storage.StatHandlerConfig{
		Redis: hiredis.Config{
			Address:  "127.0.0.1:6379",
			Net:      "tcp",
			DB:       0,
			Password: "",
			Prefix:   "hcstattest_",
			Suffix:   "_hcstattest",
		},
	}
	var healthcheckConfig = Config{
		Enable:             true,
		CheckInterval:      1,
		UpdateInterval:     200,
		MaxRequests:        20,
		MaxPendingRequests: 100,
	}

	var hcConfig = `{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.google.com.","ns":"ns1.google.com.","refresh":44,"retry":55,"expire":66}}`
	var hcEntries = [][]string{
		{"www",
			`{"a":{"ttl":300, "health_check":{"enable":true,"protocol":"http","uri":"","port":80, "up_count": 3, "down_count": -3, "timeout":1000}, "records":[{"ip":"142.250.185.68"}]}}`,
		},
		{"ddd",
			`{"a":{"ttl":300, "health_check":{"enable":true,"protocol":"http","uri":"/uri2","port":80, "up_count": 3, "down_count": -3, "timeout":1000}, "records":[{"ip":"3.3.3.3"}]}}`,
		},
		/*
			{"y",
				`{"a":{"ttl":300, "health_check":{"enable":true,"protocol":"ping", "up_count": 3, "down_count": -3, "timeout":1000}, "records":[{"ip":"4.2.2.4"}]}}`,
			},
			{"z",
				`{"a":{"ttl":300, "health_check":{"enable":true,"protocol":"ping", "up_count": 3, "down_count": -3, "timeout":1000}, "records":[{"ip":"192.168.200.2"}]}}`,
			},
		*/
	}

	log.Println("TestHealthCheck")

	dh := storage.NewDataHandler(&healthcheckRedisDataConfig)
	dh.Start()
	sh := storage.NewStatHandler(&healthcheckStatConfig)
	l, _ := zap.NewProduction()
	hc := NewHealthcheck(&healthcheckConfig, dh, sh, l)
	hc.redisStat.Clear()
	hc.redisData.Clear()
	hc.redisData.EnableZone("google.com.")
	for _, entry := range hcEntries {
		hc.redisData.SetLocationFromJson("google.com.", entry[0], entry[1])
	}
	hc.redisData.SetZoneConfigFromJson("google.com.", hcConfig)

	go hc.Start()
	time.Sleep(12 * time.Second)
	h1 := hc.redisStat.GetHealthStatus("www.google.com.", "142.250.185.68")
	h2 := hc.redisStat.GetHealthStatus("ddd.google.com.", "3.3.3.3")
	/*
		h3 := hc.getStatus("y.google.com.", net.ParseIP("4.2.2.4"))
		h4 := hc.getStatus("z.google.com.", net.ParseIP("192.168.200.2"))
	*/
	log.Println(h1, " ", h2, " " /*, h3,, " ", h4*/)
	Expect(h1).To(Equal(3))
	Expect(h2).To(Equal(-3))
	/*
		Expect(h3).To(Equal(3))
		Expect(h4).To(Equal(-3))
	*/
}

func TestExpire(t *testing.T) {
	RegisterTestingT(t)
	var statConfig = storage.StatHandlerConfig{
		Redis: hiredis.Config{
			Address:  "127.0.0.1:6379",
			Net:      "tcp",
			DB:       0,
			Password: "",
			Prefix:   "healthcheck1_",
			Suffix:   "_healthcheck1",
		},
	}
	var config = Config{
		Enable:             true,
		MaxRequests:        10,
		MaxPendingRequests: 100,
		UpdateInterval:     1,
		CheckInterval:      600,
	}

	log.Printf("TestExpire")

	dh := storage.NewDataHandler(&healthcheckRedisDataConfig)
	dh.Start()
	sh := storage.NewStatHandler(&statConfig)
	l, _ := zap.NewProduction()
	hc := NewHealthcheck(&config, dh, sh, l)

	hc.redisData.Clear()
	hc.redisStat.Clear()

	expireItem := []string{
		"w0", "1.2.3.4",
		`{"host":"w0.healthcheck.exp.", "ip":"1.2.3.4", "enable":true, "protocol":"http", "uri":"/uri0", "port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
		`{"host":"w0.healthcheck.exp.", "ip":"1.2.3.4", "enable":false, "protocol":"http", "uri":"/uri0", "port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
	}

	a := fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", expireItem[1], expireItem[2])
	log.Println(a)
	hc.redisData.EnableZone("healthcheck.exp.")
	hc.redisData.SetLocationFromJson("healthcheck.exp.", expireItem[0], a)
	var item types.HealthCheckItem
	jsoniter.Unmarshal([]byte(expireItem[2]), &item)
	hc.redisStat.SetHealthcheckItem(&item)

	go hc.Start()
	time.Sleep(time.Second * 2)
	status := hc.redisStat.GetHealthStatus("w0.healthcheck.exp.", "1.2.3.4")
	Expect(status).To(Equal(3))

	a = fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", expireItem[1], expireItem[3])
	log.Println(a)
	hc.redisData.SetLocationFromJson("healthcheck.exp.", expireItem[0], a)

	time.Sleep(time.Second * 5)
	status = hc.redisStat.GetHealthStatus("w0.healthcheck.exp.", "1.2.3.4")
	Expect(status).To(Equal(0))
}

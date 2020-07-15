package healthcheck

import (
	"fmt"
	"github.com/hawell/z42/redis"
	"github.com/hawell/z42/types"
	"log"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hawell/logger"
)

var healthcheckGetEntries = [][]string{
	{"w0.healthcheck.com.:1.2.3.4", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":3}`},
	{"w0.healthcheck.com.:2.3.4.5", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":1}`},
	{"w0.healthcheck.com.:3.4.5.6", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":0}`},
	{"w0.healthcheck.com.:4.5.6.7", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-1}`},
	{"w0.healthcheck.com.:5.6.7.8", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-3}`},

	{"w1.healthcheck.com.:2.3.4.5", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":1}`},
	{"w1.healthcheck.com.:3.4.5.6", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":0}`},
	{"w1.healthcheck.com.:4.5.6.7", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-1}`},
	{"w1.healthcheck.com.:5.6.7.8", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-3}`},

	{"w2.healthcheck.com.:3.4.5.6", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":0}`},
	{"w2.healthcheck.com.:4.5.6.7", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-1}`},
	{"w2.healthcheck.com.:5.6.7.8", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-3}`},

	{"w3.healthcheck.com.:4.5.6.7", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-1}`},
	{"w3.healthcheck.com.:5.6.7.8", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-3}`},

	{"w4.healthcheck.com.:5.6.7.8", `{"enable":true,"protocol":"http","uri":"/","port":80, "status":-3}`},
}

var stats = []int{3, 1, 0, -1, -3, 1, 0, -1, -3, 0, -1, -3, -1, -3, -3}
var filterResult = []int{1, 3, 2, 1, 1}

var healthCheckSetEntries = [][]string{
	{"@", "185.143.233.2",
		`{"enable":true,"protocol":"http","uri":"","port":80, "timeout": 1000}`,
	},
	{"www", "185.143.234.50",
		`{"enable":true,"protocol":"http","uri":"","port":80, "timeout": 1000}`,
	},
}

var healthcheckTransferItems = [][]string{
	{"w0", "1.2.3.4",
		`{"enable":true,"protocol":"http","uri":"/uri0","port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
		`{"enable":true,"protocol":"http","uri":"/uri0","port":80, "status":2, "up_count": 3, "down_count": -3, "timeout":1000}`,
	},
	{"w1", "2.3.4.5",
		`{"enable":true,"protocol":"https","uri":"/uri111","port":8081, "up_count": 3, "down_count": -3, "timeout":1000}`,
		`{"enable":true,"protocol":"http","uri":"/uri1","port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
	},
	{"w2", "3.4.5.6",
		"",
		`{"enable":true,"protocol":"http","uri":"/uri2","port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
	},
	{"w3", "4.5.6.7",
		`{"enable":true,"protocol":"http","uri":"/uri3","port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
		``,
	},
}

var healthCheckTransferResults = [][]string{
	{"w0.healthcheck.com.:1.2.3.4", `{"enable":true,"protocol":"http","uri":"/uri0","port":80, "status":2, "up_count": 3, "down_count": -3, "timeout":1000}`},
	{"w1.healthcheck.com.:2.3.4.5", `{"enable":true,"protocol":"https","uri":"/uri111","port":8081, "status":0, "up_count": 3, "down_count": -3, "timeout":1000}`},
	{"w3.healthcheck.com.:4.5.6.7", `{"enable":true,"protocol":"http","uri":"/uri3","port":80, "status":0, "up_count": 3, "down_count": -3, "timeout":1000}`},
}

var healthcheckRedisStatConfig = redis.StatHandlerConfig{
	Redis: redis.RedisConfig{
		Address:  "redis:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "healthcheck_",
		Suffix:   "_healthcheck",
		Connection: redis.RedisConnectionConfig{
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

var healthcheckRedisDataConfig = redis.DataHandlerConfig{
	ZoneCacheSize:      10000,
	ZoneCacheTimeout:   60,
	ZoneReload:         60,
	RecordCacheSize:    10000000,
	RecordCacheTimeout: 60,
	Redis: redis.RedisConfig{
		Address:  "redis:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "hcconfig_",
		Suffix:   "_hcconfig",
		Connection: redis.RedisConnectionConfig{
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

var healthcheckTestConfig = HealthcheckConfig{
	Enable:             true,
	MaxRequests:        10,
	MaxPendingRequests: 100,
	UpdateInterval:     600,
	CheckInterval:      600,
	Log: logger.LogConfig{
		Enable: true,
		Path:   "/tmp/healthcheck.log",
	},
}

func TestGet(t *testing.T) {
	log.Println("TestGet")
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)
	dh := redis.NewDataHandler(&healthcheckRedisDataConfig)
	sh := redis.NewStatHandler(&healthcheckRedisStatConfig)
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh)

	h.redisStat.Redis.Del("*")
	h.redisData.Redis.Del("*")
	for _, entry := range healthcheckGetEntries {
		h.redisStat.Redis.Set("z42:healthcheck:"+entry[0], entry[1])
	}

	for i := range healthcheckGetEntries {
		hostIp := strings.Split(healthcheckGetEntries[i][0], ":")
		stat := h.getStatus(hostIp[0], net.ParseIP(hostIp[1]))
		log.Println("[DEBUG]", stat, " ", stats[i])
		if stat != stats[i] {
			t.Fail()
		}
	}
	// h.Stop()
	h.redisStat.Redis.Del("*")
}

func TestFilter(t *testing.T) {
	log.Println("TestFilter")
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)
	dh := redis.NewDataHandler(&healthcheckRedisDataConfig)
	sh := redis.NewStatHandler(&healthcheckRedisStatConfig)
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh)

	h.redisStat.Redis.Del("*")
	h.redisData.Redis.Del("*")
	for _, entry := range healthcheckGetEntries {
		h.redisStat.Redis.Set("z42:healthcheck:"+entry[0], entry[1])
	}

	w := []types.Record{
		{
			RRSets: types.RRSets{
				A: types.IP_RRSet{
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
			},
		},
		{
			RRSets: types.RRSets{
				A: types.IP_RRSet{
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
			},
		},
		{
			RRSets: types.RRSets{
				A: types.IP_RRSet{
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
			},
		},
		{
			RRSets: types.RRSets{
				A: types.IP_RRSet{
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
			},
		},
		{
			RRSets: types.RRSets{
				A: types.IP_RRSet{
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
			},
		},
	}
	for i := range w {
		log.Println("[DEBUG]", w[i])
		mask := make([]int, len(w[i].A.Data))
		mask = h.FilterHealthcheck("w"+strconv.Itoa(i)+".healthcheck.com.", &w[i].A, mask)
		log.Println("[DEBUG]", w[i])
		count := 0
		for _, x := range mask {
			if x == types.IpMaskWhite {
				count++
			}
		}
		if count != filterResult[i] {
			t.Fail()
		}
	}
	h.redisStat.Redis.Del("*")
	// h.Stop()
}

func TestSet(t *testing.T) {
	log.Println("TestSet")
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)
	dh := redis.NewDataHandler(&healthcheckRedisDataConfig)
	sh := redis.NewStatHandler(&healthcheckRedisStatConfig)
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh)

	h.redisStat.Redis.Del("*")
	h.redisData.Redis.Del("*")
	for _, str := range healthCheckSetEntries {
		a := fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", str[1], str[2])
		h.redisStat.Redis.HSet("z42:zones:healthcheck.com.", str[0], a)
		var key string
		if str[0] == "@" {
			key = fmt.Sprintf("example.com.:%s", str[1])
		} else {
			key = fmt.Sprintf("%s.example.com.:%s", str[0], str[1])
		}
		h.redisStat.Redis.Set("z42:healthcheck:"+key, str[2])
	}
	// h.transferItems()
	go h.Start()
	time.Sleep(time.Second * 10)

	log.Println("[DEBUG]", h.getStatus("example.com", net.ParseIP("185.143.233.2")))
	log.Println("[DEBUG]", h.getStatus("www.example.com", net.ParseIP("185.143.234.50")))
}

func TestTransfer(t *testing.T) {
	log.Printf("TestTransfer")
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)
	dh := redis.NewDataHandler(&healthcheckRedisDataConfig)
	sh := redis.NewStatHandler(&healthcheckRedisStatConfig)
	h := NewHealthcheck(&healthcheckTestConfig, dh, sh)

	h.redisData.Redis.Del("*")
	h.redisStat.Redis.Del("*")
	h.redisData.Redis.SAdd("z42:zones", "healthcheck.com.")
	for _, str := range healthcheckTransferItems {
		if str[2] != "" {
			a := fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", str[1], str[2])
			h.redisData.Redis.HSet("z42:zones:healthcheck.com.", str[0], a)
		}
		if str[3] != "" {
			key := fmt.Sprintf("%s.healthcheck.com.:%s", str[0], str[1])
			h.redisStat.Redis.Set("z42:healthcheck:"+key, str[3])
		}
	}

	// h.transferItems()
	go h.Start()
	time.Sleep(time.Second * 10)

	itemsEqual := func(item1 *HealthCheckItem, item2 *HealthCheckItem) bool {
		if item1.Ip != item2.Ip || item1.Uri != item2.Uri || item1.Port != item2.Port ||
			item1.Protocol != item2.Protocol || item1.Enable != item2.Enable ||
			item1.UpCount != item2.UpCount || item1.DownCount != item2.DownCount || item1.Timeout != item2.Timeout {
			return false
		}
		return true
	}

	for i, str := range healthCheckTransferResults {
		h.redisStat.Redis.Set("z42:healthcheck:"+str[0]+"res", str[1])
		resItem := h.loadItem(str[0] + "res")
		resItem.Ip = strings.TrimRight(resItem.Ip, "res")
		storedItem := h.loadItem(str[0])
		log.Println("** key : ", str[0])
		log.Println("** expected : ", resItem)
		log.Println("** stored : ", storedItem)
		if !itemsEqual(resItem, storedItem) {
			log.Println(i, "failed")
			t.Fail()
		}
	}
}

/*
func TestPing(t *testing.T) {
	log.Println("TestPing")
	if err := pingCheck("4.2.2.4", time.Second); err != nil {
		t.Fail()
	}
}
*/

func TestHealthCheck(t *testing.T) {
	var healthcheckStatConfig = redis.StatHandlerConfig{
		Redis: redis.RedisConfig{
			Address:  "redis:6379",
			Net:      "tcp",
			DB:       0,
			Password: "",
			Prefix:   "hcstattest_",
			Suffix:   "_hcstattest",
		},
	}
	var healthcheckConfig = HealthcheckConfig{
		Enable: true,
		Log: logger.LogConfig{
			Enable:     true,
			Target:     "file",
			Level:      "info",
			Path:       "/tmp/hctest.log",
			TimeFormat: "2006-01-02 15:04:05",
		},
		CheckInterval:      1,
		UpdateInterval:     200,
		MaxRequests:        20,
		MaxPendingRequests: 100,
	}

	var hcConfig = `{"soa":{"ttl":300, "minttl":100, "mbox":"hostmaster.google.com.","ns":"ns1.google.com.","refresh":44,"retry":55,"expire":66}}`
	var hcEntries = [][]string{
		{"www",
			`{"a":{"ttl":300, "health_check":{"enable":true,"protocol":"http","uri":"","port":80, "up_count": 3, "down_count": -3, "timeout":1000}, "records":[{"ip":"172.217.17.78"}]}}`,
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
	logger.Default = logger.NewLogger(&logger.LogConfig{Enable: true, Target: "stdout", Format: "text"}, nil)

	dh := redis.NewDataHandler(&healthcheckRedisDataConfig)
	sh := redis.NewStatHandler(&healthcheckStatConfig)
	hc := NewHealthcheck(&healthcheckConfig, dh, sh)
	hc.redisStat.Redis.Del("*")
	hc.redisData.Redis.Del("*")
	hc.redisData.Redis.SAdd("z42:zones", "google.com.")
	for _, entry := range hcEntries {
		hc.redisData.Redis.HSet("z42:zones:google.com.", entry[0], entry[1])
	}
	hc.redisData.Redis.Set("z42:zones:google.com.:config", hcConfig)

	go hc.Start()
	time.Sleep(12 * time.Second)
	h1 := hc.getStatus("www.google.com.", net.ParseIP("172.217.17.78"))
	h2 := hc.getStatus("ddd.google.com.", net.ParseIP("3.3.3.3"))
	/*
		h3 := hc.getStatus("y.google.com.", net.ParseIP("4.2.2.4"))
		h4 := hc.getStatus("z.google.com.", net.ParseIP("192.168.200.2"))
	*/
	log.Println(h1, " ", h2, " " /*, h3,, " ", h4*/)
	if h1 != 3 {
		t.Fail()
	}
	if h2 != -3 {
		t.Fail()
	}
	/*
	   if h3 != 3 {
	       t.Fail()
	   }
	   if h4 != -3 {
	       t.Fail()
	   }
	*/
}

func TestExpire(t *testing.T) {
	var statConfig = redis.StatHandlerConfig{
		Redis: redis.RedisConfig{
			Address:  "redis:6379",
			Net:      "tcp",
			DB:       0,
			Password: "",
			Prefix:   "healthcheck1_",
			Suffix:   "_healthcheck1",
		},
	}
	var config = HealthcheckConfig{
		Enable:             true,
		MaxRequests:        10,
		MaxPendingRequests: 100,
		UpdateInterval:     1,
		CheckInterval:      600,
		Log: logger.LogConfig{
			Enable: true,
			Path:   "/tmp/healthcheck.log",
		},
	}

	log.Printf("TestExpire")
	logger.Default = logger.NewLogger(&logger.LogConfig{}, nil)

	dh := redis.NewDataHandler(&healthcheckRedisDataConfig)
	sh := redis.NewStatHandler(&statConfig)
	hc := NewHealthcheck(&config, dh, sh)

	hc.redisData.Redis.Del("*")
	hc.redisStat.Redis.Del("*")

	expireItem := []string{
		"w0", "1.2.3.4",
		`{"enable":true,"protocol":"http","uri":"/uri0","port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
		`{"enable":false,"protocol":"http","uri":"/uri0","port":80, "status":3, "up_count": 3, "down_count": -3, "timeout":1000}`,
	}

	a := fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", expireItem[1], expireItem[2])
	log.Println(a)
	hc.redisData.Redis.SAdd("z42:zones", "healthcheck.exp.")
	hc.redisData.Redis.HSet("z42:zones:healthcheck.exp.", expireItem[0], a)
	key := fmt.Sprintf("%s.healthcheck.exp.:%s", expireItem[0], expireItem[1])
	hc.redisStat.Redis.Set("z42:healthcheck:"+key, expireItem[2])

	go hc.Start()
	time.Sleep(time.Second * 2)
	status := hc.getStatus("w0.healthcheck.exp.", net.ParseIP("1.2.3.4"))
	if status != 3 {
		fmt.Println("1")
		t.Fail()
	}

	a = fmt.Sprintf("{\"a\":{\"ttl\":300, \"records\":[{\"ip\":\"%s\"}],\"health_check\":%s}}", expireItem[1], expireItem[3])
	log.Println(a)
	hc.redisData.Redis.HSet("z42:zones:healthcheck.exp.", expireItem[0], a)

	time.Sleep(time.Second * 5)
	status = hc.getStatus("w0.healthcheck.exp.", net.ParseIP("1.2.3.4"))
	if status != 0 {
		fmt.Println("2")
		t.Fail()
	}
}

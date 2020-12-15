package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/hiredis"
	"github.com/miekg/dns"
	"math/rand"
	"os"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)
const suffix = ".tst."

var src = rand.NewSource(time.Now().UnixNano())

func RandomString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func main() {
	zonesPtr := flag.Int("zones", 10, "number of zones")
	entriesPtr := flag.Int("entries", 100, "number of entries per zone")
	typePtr := flag.String("type", "a", "record type")
	// missChancePtr := flag.Int("miss", 30, "miss chance")

	redisAddrPtr := flag.String("addr", "localhost:6379", "redis address")
	// redisAuthPtr := flag.String("auth", "foobared", "redis authentication")

	flag.Parse()

	// opts := []redis.DialOption{}
	// opts = append(opts, redis.DialPassword(*redisAuthPtr))
	dh := storage.NewDataHandler(&storage.DataHandlerConfig{
		ZoneCacheSize:      10000,
		ZoneCacheTimeout:   60,
		ZoneReload:         1,
		RecordCacheSize:    1000000,
		RecordCacheTimeout: 60,
		Redis: hiredis.Config{
			Suffix:  "",
			Prefix:  "",
			Address: *redisAddrPtr,
			Net:     "tcp",
			DB:      0,
			Connection: hiredis.ConnectionConfig{
				MaxIdleConnections:   10,
				MaxActiveConnections: 10,
				ConnectTimeout:       600,
				ReadTimeout:          600,
				IdleKeepAlive:        6000,
				MaxKeepAlive:         6000,
				WaitForConnection:    true,
			},
		},
	})

	dh.Clear()

	fq, err := os.Create("../query.txt")
	if err != nil {
		fmt.Println("cannot open file query.txt")
		return
	}
	defer fq.Close()
	wq := bufio.NewWriter(fq)

	for i := 0; i < *zonesPtr; i++ {
		fmt.Println("zone :", i)
		zoneName := RandomString(15) + suffix
		fz, err := os.Create("../" + zoneName)
		if err != nil {
			fmt.Println("cannot open file "+zoneName, " : ", err)
			return
		}
		dh.EnableZone(zoneName)
		wz := bufio.NewWriter(fz)
		wz.WriteString("$ORIGIN " + zoneName + "\n" +
			"$TTL 300\n\n" +
			"@       SOA ns1 hostmaster (\n" +
			"1      ; serial\n" +
			"7200   ; refresh\n" +
			"30M    ; retry\n" +
			"3D     ; expire\n" +
			"900    ; ncache\n" +
			")\n" +
			"@ NS ns1." + zoneName + "\n" +
			"ns1 A 1.2.3.4\n\n")

		for j := 0; j < *entriesPtr; j++ {

			fmt.Println("record :", j)
			switch *typePtr {
			case "cname":
				location1 := RandomString(15)
				location2 := RandomString(15)

				dh.SetLocationFromJson(zoneName, location1, `{"cname":{"ttl":300, "host":"`+location2+"."+zoneName+`."}}`)

				ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
				dh.SetLocationFromJson(zoneName, location2, `{"a":{"ttl":300, "records":[{"ip":"`+ip+`"}]}}`)

				wq.WriteString(fmt.Sprintf("%s.%s %d %d %s\n", location1, zoneName, dns.TypeA, dns.RcodeSuccess, ip))

				wz.WriteString(location1 + " CNAME " + location2 + "\n")
				wz.WriteString(location2 + " A " + ip + "\n")

			case "txt":
				location := RandomString(15)
				txt := RandomString(200)

				dh.SetLocationFromJson(zoneName, location, `{"txt":{"ttl":300, "records:{"text":"`+txt+`"}"}}`)

				wq.WriteString(fmt.Sprintf("%s.%s %d %d %s\n", location, zoneName, dns.TypeTXT, dns.RcodeSuccess, txt))
				wz.WriteString(location + ` TXT "` + txt + `"`)

			case "nxdomain":
				location := RandomString(15)
				wq.WriteString(fmt.Sprintf("%s.%s %d %d\n", location, zoneName, dns.TypeA, dns.RcodeNameError))

			case "a":
				fallthrough
			default:
				location := RandomString(15)

				ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))

				dh.SetLocationFromJson(zoneName, location, `{"a":{"ttl":300, "records":[{"ip":"`+ip+`"}]}}`)

				wq.WriteString(fmt.Sprintf("%s.%s %d %d %s\n", location, zoneName, dns.TypeA, dns.RcodeSuccess, ip))
				wz.WriteString(location + " A " + ip + "\n")
			}
		}
		wz.Flush()
		fz.Close()
	}
	wq.Flush()
}

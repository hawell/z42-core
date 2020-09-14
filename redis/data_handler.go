package redis

import (
	"github.com/dgraph-io/ristretto"
	"github.com/hashicorp/go-immutable-radix"
	"github.com/hawell/logger"
	"github.com/hawell/z42/dnssec"
	"github.com/hawell/z42/types"
	"github.com/json-iterator/go"
	"github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
	"strings"
	"sync"
	"time"
)

type DataHandlerConfig struct {
	ZoneCacheSize      int         `json:"zone_cache_size"`
	ZoneCacheTimeout   int         `json:"zone_cache_timeout"`
	ZoneReload         int         `json:"zone_reload"`
	RecordCacheSize    int         `json:"record_cache_size"`
	RecordCacheTimeout int         `json:"record_cache_timeout"`
	Redis              RedisConfig `json:"redis"`
}

type DataHandler struct {
	Redis              *Redis
	Zones              *iradix.Tree
	LastZoneUpdate     time.Time
	RecordCache        *ristretto.Cache
	RecordInflight     *singleflight.Group
	recordCacheTimeout int64
	ZoneCache          *ristretto.Cache
	ZoneInflight       *singleflight.Group
	zoneCacheTimeout   int64
	quit               chan struct{}
	quitWG             sync.WaitGroup
}

const (
	ZoneForcedReload = time.Minute * 60
)

func NewDataHandler(config *DataHandlerConfig) *DataHandler {
	dh := &DataHandler{
		Redis:              NewRedis(&config.Redis),
		Zones:              iradix.New(),
		RecordInflight:     new(singleflight.Group),
		ZoneInflight:       new(singleflight.Group),
		quit:               make(chan struct{}),
		recordCacheTimeout: int64(config.RecordCacheTimeout),
		zoneCacheTimeout:   int64(config.ZoneCacheTimeout),
	}
	dh.ZoneCache, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(config.ZoneCacheSize) * 10,
		MaxCost:     int64(config.ZoneCacheSize),
		BufferItems: 64,
		Metrics:     false,
	})
	dh.RecordCache, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(config.RecordCacheSize) * 10,
		MaxCost:     int64(config.RecordCacheSize),
		BufferItems: 64,
		Metrics:     false,
	})

	dh.LoadZones()

	go func() {
		// logger.Default.Debug("zone updater")
		dh.quitWG.Add(1)
		quit := make(chan *sync.WaitGroup, 1)
		modified := false
		go dh.Redis.SubscribeEvent("z42:zones", func() {
			modified = true
		}, func(channel string, data string) {
			modified = true
		}, func(err error) {
			logger.Default.Error(err)
		}, quit)

		reloadTicker := time.NewTicker(time.Duration(config.ZoneReload) * time.Second)
		forceReloadTicker := time.NewTicker(ZoneForcedReload)
		for {
			select {
			case <-dh.quit:
				reloadTicker.Stop()
				forceReloadTicker.Stop()
				// logger.Default.Debug("zone updater stopped")
				quit <- &dh.quitWG
				return
			case <-reloadTicker.C:
				if modified {
					// logger.Default.Debug("loading zones")
					dh.LoadZones()
					modified = false
				}
			case <-forceReloadTicker.C:
				modified = true
			}
		}
	}()

	return dh
}

func (dh *DataHandler) ShutDown() {
	close(dh.quit)
	dh.quitWG.Wait()
}

func (dh *DataHandler) LoadZones() {
	dh.LastZoneUpdate = time.Now()
	zones, err := dh.Redis.SMembers("z42:zones")
	if err != nil {
		logger.Default.Error("cannot load zones : ", err)
		return
	}
	newZones := iradix.New()
	for _, zone := range zones {
		newZones, _, _ = newZones.Insert(types.ReverseName(zone), zone)
	}
	dh.Zones = newZones
}

func (dh *DataHandler) FindZone(qname string) string {
	rname := types.ReverseName(qname)
	if _, zname, ok := dh.Zones.Root().LongestPrefix(rname); ok {
		return zname.(string)
	}
	return ""
}

func (dh *DataHandler) GetZone(zone string) *types.Zone {
	cachedZone, found := dh.ZoneCache.Get(zone)
	var z *types.Zone = nil
	if found && cachedZone != nil {
		z = cachedZone.(*types.Zone)
		if time.Now().Unix() <= z.CacheTimeout {
			return z
		}
	}

	answer, _, _ := dh.ZoneInflight.Do(zone, func() (interface{}, error) {
		locations, err := dh.Redis.GetHKeys("z42:zones:" + zone)
		if err != nil {
			logger.Default.Errorf("cannot load zone %s locations : %s", zone, err)
			return nil, err
		}
		config, err := dh.Redis.Get("z42:zones:" + zone + ":config")
		if err != nil {
			logger.Default.Errorf("cannot load zone %s config : %s", zone, err)
		}

		z := types.NewZone(zone, locations, config)
		dh.loadZoneKeys(z)
		z.CacheTimeout = time.Now().Unix() + dh.zoneCacheTimeout

		dh.ZoneCache.Set(zone, z, 1)
		return z, nil
	})
	if answer != nil {
		return answer.(*types.Zone)
	}
	return z
}

func (dh *DataHandler) loadKey(pub string, priv string) *types.ZoneKey {
	pubStr, _ := dh.Redis.Get(pub)
	if pubStr == "" {
		logger.Default.Errorf("key is not set : %s", pub)
		return nil
	}
	privStr, _ := dh.Redis.Get(priv)
	if privStr == "" {
		logger.Default.Errorf("key is not set : %s", priv)
		return nil
	}
	privStr = strings.Replace(privStr, "\\n", "\n", -1)
	zoneKey := new(types.ZoneKey)
	if rr, err := dns.NewRR(pubStr); err == nil {
		zoneKey.DnsKey = rr.(*dns.DNSKEY)
	} else {
		logger.Default.Errorf("cannot parse zone key : %s", err)
		return nil
	}
	if pk, err := zoneKey.DnsKey.NewPrivateKey(privStr); err == nil {
		zoneKey.PrivateKey = pk
	} else {
		logger.Default.Errorf("cannot create private key : %s", err)
		return nil
	}
	now := time.Now()
	zoneKey.KeyInception = uint32(now.Add(-3 * time.Hour).Unix())
	zoneKey.KeyExpiration = uint32(now.Add(8 * 24 * time.Hour).Unix())
	return zoneKey
}

func (dh *DataHandler) loadZoneKeys(z *types.Zone) {
	if z.Config.DnsSec {
		z.ZSK = dh.loadKey("z42:zones:"+z.Name+":zsk:pub", "z42:zones:"+z.Name+":zsk:priv")
		if z.ZSK == nil {
			z.Config.DnsSec = false
			return
		}
		z.KSK = dh.loadKey("z42:zones:"+z.Name+":ksk:pub", "z42:zones:"+z.Name+":ksk:priv")
		if z.KSK == nil {
			z.Config.DnsSec = false
			return
		}

		z.ZSK.DnsKey.Flags = 256
		z.KSK.DnsKey.Flags = 257
		if z.ZSK.DnsKey.Hdr.Ttl != z.KSK.DnsKey.Hdr.Ttl {
			z.ZSK.DnsKey.Hdr.Ttl = z.KSK.DnsKey.Hdr.Ttl
		}

		if rrsig := dnssec.SignRRSet([]dns.RR{z.ZSK.DnsKey, z.KSK.DnsKey}, z.Name, z.KSK, z.KSK.DnsKey.Hdr.Ttl); rrsig != nil {
			z.DnsKeySig = rrsig
		} else {
			logger.Default.Errorf("cannot create RRSIG for DNSKEY")
			z.Config.DnsSec = false
			return
		}
	}
}

func (dh *DataHandler) GetLocation(location string, z *types.Zone) *types.Record {
	key := location + "." + z.Name
	var r *types.Record = nil
	cachedRecord, found := dh.RecordCache.Get(key)
	if found && cachedRecord != nil {
		r = cachedRecord.(*types.Record)
		if time.Now().Unix() <= r.CacheTimeout {
			return r
		}
	}

	answer, _, _ := dh.RecordInflight.Do(key, func() (interface{}, error) {
		var label, name string
		if location == z.Name {
			name = z.Name
			label = "@"
		} else {
			name = location + "." + z.Name
			label = location
		}
		r := new(types.Record)
		r.A = types.IP_RRSet{
			FilterConfig: types.IpFilterConfig{
				Count:     "multi",
				Order:     "none",
				GeoFilter: "none",
			},
			HealthCheckConfig: types.IpHealthCheckConfig{
				Enable: false,
			},
		}
		r.AAAA = r.A
		r.Name = name

		val, err := dh.Redis.HGet("z42:zones:"+z.Name, label)
		if err != nil {
			if label == "@" {
				dh.RecordCache.Set(key, r, 1)
				return r, nil
			}
			logger.Default.Error(err, " : ", label, " ", z.Name)
			return nil, err
		}
		if val != "" {
			err := jsoniter.Unmarshal([]byte(val), r)
			if err != nil {
				logger.Default.Errorf("cannot parse json : zone -> %s, location -> %s, \"%s\" -> %s", z.Name, location, val, err)
				return nil, err
			}
		}
		r.CacheTimeout = time.Now().Unix() + dh.recordCacheTimeout
		dh.RecordCache.Set(key, r, 1)
		return r, nil
	})

	if answer != nil {
		return answer.(*types.Record)
	}
	return r
}

func (dh *DataHandler) SetLocation(location string, z *types.Zone, val *types.Record) {
	jsonValue, err := jsoniter.Marshal(val)
	if err != nil {
		logger.Default.Errorf("cannot encode to json : %s", err)
		return
	}
	var label string
	if location == z.Name {
		label = "@"
	} else {
		label = location
	}
	if err = dh.Redis.HSet(z.Name, label, string(jsonValue)); err != nil {
		logger.Default.Error("redis error : ", err)
	}
}

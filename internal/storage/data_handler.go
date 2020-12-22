package storage

import (
	"errors"
	"github.com/dgraph-io/ristretto"
	"github.com/hashicorp/go-immutable-radix"
	"github.com/hawell/z42/internal/dnssec"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/pkg/hiredis"
	"github.com/json-iterator/go"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"strings"
	"sync"
	"time"
)

type DataHandlerConfig struct {
	ZoneCacheSize      int            `json:"zone_cache_size"`
	ZoneCacheTimeout   int            `json:"zone_cache_timeout"`
	ZoneReload         int            `json:"zone_reload"`
	RecordCacheSize    int            `json:"record_cache_size"`
	RecordCacheTimeout int            `json:"record_cache_timeout"`
	Redis              hiredis.Config `json:"redis"`
}

type DataHandler struct {
	redis              *hiredis.Redis
	zones              *iradix.Tree
	lastZoneUpdate     time.Time
	recordCache        *ristretto.Cache
	recordInflight     *singleflight.Group
	recordCacheTimeout int64
	zoneCache          *ristretto.Cache
	zoneInflight       *singleflight.Group
	zoneCacheTimeout   int64
	quit               chan struct{}
	quitWG             sync.WaitGroup
}

const (
	zoneForcedReload = time.Minute * 60
	keyPrefix        = "z42:zones:"
	zonesKey         = "z42:zones"
)

func NewDataHandler(config *DataHandlerConfig) *DataHandler {
	dh := &DataHandler{
		redis:              hiredis.NewRedis(&config.Redis),
		zones:              iradix.New(),
		recordInflight:     new(singleflight.Group),
		zoneInflight:       new(singleflight.Group),
		quit:               make(chan struct{}),
		recordCacheTimeout: int64(config.RecordCacheTimeout),
		zoneCacheTimeout:   int64(config.ZoneCacheTimeout),
	}
	dh.zoneCache, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(config.ZoneCacheSize) * 10,
		MaxCost:     int64(config.ZoneCacheSize),
		BufferItems: 64,
		Metrics:     false,
	})
	dh.recordCache, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(config.RecordCacheSize) * 10,
		MaxCost:     int64(config.RecordCacheSize),
		BufferItems: 64,
		Metrics:     false,
	})

	dh.LoadZones()

	go func() {
		zap.L().Debug("zone updater")
		dh.quitWG.Add(1)
		zoneListQuitChan := make(chan *sync.WaitGroup, 1)
		modified := false
		go dh.redis.SubscribeEvent(zonesKey, func() {
			modified = true
		}, func(channel string, data string) {
			modified = true
		}, func(err error) {
			zap.L().Error("error", zap.Error(err))
		}, zoneListQuitChan)

		dh.quitWG.Add(1)
		zonesQuitChan := make(chan *sync.WaitGroup, 1)
		go dh.redis.SubscribeEvent(keyPrefix+"*", func() {
		}, func(channel string, data string) {
			keyStr := strings.TrimPrefix(channel, keyPrefix)
			keyParts := splitDbKey(keyStr)
			if zone, label, ok := isLocationEntry(keyParts); ok {
				dh.recordCache.Del(label + "." + zone)
			} else if zone, ok := isConfigEntry(keyParts); ok {
				dh.zoneCache.Del(zone)
			}
		}, func(err error) {
			zap.L().Error("error", zap.Error(err))
		}, zonesQuitChan)

		reloadTicker := time.NewTicker(time.Duration(config.ZoneReload) * time.Second)
		forceReloadTicker := time.NewTicker(zoneForcedReload)
		for {
			select {
			case <-dh.quit:
				reloadTicker.Stop()
				forceReloadTicker.Stop()
				zap.L().Debug("zone updater stopped")
				zoneListQuitChan <- &dh.quitWG
				zonesQuitChan <- &dh.quitWG
				return
			case <-reloadTicker.C:
				if modified {
					zap.L().Debug("loading zones")
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

func isConfigEntry(parts []string) (string, bool) {
	if len(parts) == 2 && parts[1] == "config" {
		return parts[0], true
	}
	return "", false
}

func isLocationEntry(parts []string) (string, string, bool) {
	if len(parts) == 3 && parts[1] == "labels" {
		return parts[0], parts[2], true
	}
	return "", "", false
}

func splitDbKey(key string) []string {
	return strings.Split(key, ":")
}

func (dh *DataHandler) ShutDown() {
	close(dh.quit)
	dh.quitWG.Wait()
}

func zoneConfigKey(zone string) string {
	return keyPrefix + zone + ":config"
}

func zoneLocationKey(zone string, label string) string {
	return keyPrefix + zone + ":labels:" + label
}

func zonePubKey(zone string, keyType string) string {
	return keyPrefix + zone + ":" + keyType + ":pub"
}

func zonePrivKey(zone string, keyType string) string {
	return keyPrefix + zone + ":" + keyType + ":priv"
}

// TODO: make this function internal
func (dh *DataHandler) LoadZones() {
	dh.lastZoneUpdate = time.Now()
	zones, err := dh.redis.SMembers(zonesKey)
	if err != nil {
		zap.L().Error("cannot load zones", zap.Error(err))
		return
	}
	newZones := iradix.New()
	for _, zone := range zones {
		newZones, _, _ = newZones.Insert(types.ReverseName(zone), zone)
	}
	dh.zones = newZones
}

func (dh *DataHandler) EnableZone(zone string) error {
	return dh.redis.SAdd(zonesKey, zone)
}

func (dh *DataHandler) DisableZone(zone string) error {
	return dh.redis.SRem(zonesKey, zone)
}

func (dh *DataHandler) FindZone(qname string) string {
	rname := types.ReverseName(qname)
	if _, zname, ok := dh.zones.Root().LongestPrefix(rname); ok {
		return zname.(string)
	}
	return ""
}

func (dh *DataHandler) GetZone(zone string) *types.Zone {
	cachedZone, found := dh.zoneCache.Get(zone)
	var z *types.Zone = nil
	if found && cachedZone != nil {
		z = cachedZone.(*types.Zone)
		if time.Now().Unix() <= z.CacheTimeout {
			return z
		}
	}

	answer, _, _ := dh.zoneInflight.Do(zone, func() (interface{}, error) {
		locations, err := dh.redis.GetKeys(zoneLocationKey(zone, "*"))
		if err != nil {
			zap.L().Error("cannot load zone locations", zap.String("zone", zone), zap.Error(err))
			return nil, err
		}
		trm := zoneLocationKey(zone, "")
		for i, s := range locations {
			locations[i] = strings.TrimPrefix(s, trm)
		}

		configStr, err := dh.redis.Get(zoneConfigKey(zone))
		if err != nil {
			zap.L().Error("cannot load zone config", zap.String("zone", zone), zap.Error(err))
		}

		z := types.NewZone(zone, locations, configStr)
		dh.loadZoneKeys(z)
		z.CacheTimeout = time.Now().Unix() + dh.zoneCacheTimeout

		dh.zoneCache.Set(zone, z, 1)
		return z, nil
	})
	if answer != nil {
		return answer.(*types.Zone)
	}
	return z
}

func (dh *DataHandler) GetZoneConfig(zone string) (*types.ZoneConfig, error) {
	z := dh.GetZone(zone)
	if z == nil {
		return nil, errors.New("cannot get zone")
	}
	return z.Config, nil
}

func (dh *DataHandler) GetZones() []string {
	domains, err := dh.redis.SMembers(zonesKey)
	if err != nil {
		zap.L().Error("cannot get zone list", zap.String("key", zonesKey), zap.Error(err))
		return nil
	}
	return domains
}

func (dh *DataHandler) GetZoneLocations(zone string) []string {
	z := dh.GetZone(zone)
	if z == nil {
		return nil
	}
	return z.LocationsList
}

func (dh *DataHandler) SetZoneConfig(zone string, config *types.ZoneConfig) error {
	json, err := jsoniter.Marshal(config)
	if err != nil {
		return err
	}
	return dh.SetZoneConfigFromJson(zone, string(json))
}

func (dh *DataHandler) SetZoneConfigFromJson(zone string, config string) error {
	return dh.redis.Set(zoneConfigKey(zone), config)
}

func (dh *DataHandler) GetLocation(zone string, label string) (*types.Record, error) {
	key := label + "." + zone
	var r *types.Record = nil
	cachedRecord, found := dh.recordCache.Get(key)
	if found && cachedRecord != nil {
		r = cachedRecord.(*types.Record)
		if time.Now().Unix() <= r.CacheTimeout {
			return r, nil
		}
	}

	answer, err, _ := dh.recordInflight.Do(key, func() (interface{}, error) {
		r := new(types.Record)
		r.CacheTimeout = time.Now().Unix() + dh.recordCacheTimeout

		val, err := dh.redis.Get(zoneLocationKey(zone, label))
		if err != nil {
			if label == "@" {
				dh.recordCache.Set(key, r, 1)
				return r, nil
			}
			zap.L().Error("cannot get location", zap.Error(err), zap.String("label", label), zap.String("zone", zone))
			return nil, err
		}
		if val != "" {
			err := jsoniter.Unmarshal([]byte(val), r)
			if err != nil {
				zap.L().Error(
					"cannot parse json",
					zap.String("zone", zone),
					zap.String("label", label),
					zap.String("json", val),
					zap.Error(err),
				)
				return nil, err
			}
		}
		dh.recordCache.Set(key, r, 1)
		return r, nil
	})

	if answer == nil {
		return nil, err
	}
	return answer.(*types.Record), nil
}

func (dh *DataHandler) SetLocation(zone string, label string, val *types.Record) error {
	jsonValue, err := jsoniter.Marshal(val)
	if err != nil {
		return err
	}
	return dh.SetLocationFromJson(zone, label, string(jsonValue))
}

func (dh *DataHandler) SetLocationFromJson(zone string, label string, val string) error {
	return dh.redis.Set(zoneLocationKey(zone, label), val)
}

func (dh *DataHandler) RemoveLocation(zone string, label string) error {
	return dh.redis.Del(zoneLocationKey(zone, label))
}

func (dh *DataHandler) SetZoneKey(zone string, keyType string, pub string, priv string) error {
	if err := dh.redis.Set(zonePubKey(zone, keyType), pub); err != nil {
		return err
	}
	return dh.redis.Set(zonePrivKey(zone, keyType), priv)
}

func (dh *DataHandler) loadKey(zone string, keyType string) *types.ZoneKey {
	pubStr, _ := dh.redis.Get(zonePubKey(zone, keyType))
	if pubStr == "" {
		zap.L().Error("key is not set", zap.String("key", zonePubKey(zone, keyType)))
		return nil
	}
	privStr, _ := dh.redis.Get(zonePrivKey(zone, keyType))
	if privStr == "" {
		zap.L().Error("key is not set", zap.String("key", zonePrivKey(zone, keyType)))
		return nil
	}
	privStr = strings.Replace(privStr, "\\n", "\n", -1)
	zoneKey := new(types.ZoneKey)
	if rr, err := dns.NewRR(pubStr); err == nil {
		zoneKey.DnsKey = rr.(*dns.DNSKEY)
	} else {
		zap.L().Error("cannot parse zone key", zap.Error(err))
		return nil
	}
	if pk, err := zoneKey.DnsKey.NewPrivateKey(privStr); err == nil {
		zoneKey.PrivateKey = pk
	} else {
		zap.L().Error("cannot create private key", zap.Error(err))
		return nil
	}
	now := time.Now()
	zoneKey.KeyInception = uint32(now.Add(-3 * time.Hour).Unix())
	zoneKey.KeyExpiration = uint32(now.Add(8 * 24 * time.Hour).Unix())
	return zoneKey
}

func (dh *DataHandler) loadZoneKeys(z *types.Zone) {
	if z.Config.DnsSec {
		z.ZSK = dh.loadKey(z.Name, "zsk")
		if z.ZSK == nil {
			z.Config.DnsSec = false
			return
		}
		z.KSK = dh.loadKey(z.Name, "ksk")
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
			zap.L().Error("cannot create RRSIG for DNSKEY")
			z.Config.DnsSec = false
			return
		}
	}
}

func (dh *DataHandler) Clear() error {
	return dh.redis.Del("*")
}

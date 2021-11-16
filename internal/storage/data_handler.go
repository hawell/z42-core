package storage

import (
	"errors"
	"fmt"
	"github.com/dgraph-io/ristretto"
	redisCon "github.com/gomodule/redigo/redis"
	"github.com/hashicorp/go-immutable-radix"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/dnssec"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/pkg/hiredis"
	"github.com/json-iterator/go"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DataHandlerConfig struct {
	ZoneCacheSize      int            `json:"zone_cache_size"`
	ZoneCacheTimeout   int64          `json:"zone_cache_timeout"`
	ZoneReload         int            `json:"zone_reload"`
	RecordCacheSize    int            `json:"record_cache_size"`
	RecordCacheTimeout int64          `json:"record_cache_timeout"`
	Redis              hiredis.Config `json:"redis"`
	MinTTL             uint32         `json:"min_ttl,default:5"`
	MaxTTL             uint32         `json:"max_ttl,default:3600"`
}

type DataHandler struct {
	config         *DataHandlerConfig
	redis          *hiredis.Redis
	zones          *iradix.Tree
	lastZoneUpdate time.Time
	recordCache    *ristretto.Cache
	recordInflight *singleflight.Group
	zoneCache      *ristretto.Cache
	zoneInflight   *singleflight.Group
	quit           chan struct{}
	quitWG         sync.WaitGroup
}

const (
	zoneForcedReload = time.Minute * 60
	keyPrefix        = "z42:zones:"
	zonesKey         = "z42:zones"
	revisionKey      = "z42:revision"
)

func NewDataHandler(config *DataHandlerConfig) *DataHandler {
	dh := &DataHandler{
		config:         config,
		redis:          hiredis.NewRedis(&config.Redis),
		zones:          iradix.New(),
		recordInflight: new(singleflight.Group),
		zoneInflight:   new(singleflight.Group),
		quit:           make(chan struct{}),
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

	return dh
}

func (dh *DataHandler) Start() {
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
			keyStr := channel
			keyParts := splitDbKey(keyStr)
			if isRRSetEntry(keyParts) {
				dh.recordCache.Del(keyStr)
			} else {
				dh.zoneCache.Del(keyParts[0])
			}
		}, func(err error) {
			zap.L().Error("error", zap.Error(err))
		}, zonesQuitChan)

		reloadTicker := time.NewTicker(time.Duration(dh.config.ZoneReload) * time.Second)
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
}

func isRRSetEntry(parts []string) bool {
	if len(parts) == 4 && parts[1] == "labels" {
		return true
	}
	return false
}

func splitDbKey(key string) []string {
	key = strings.TrimPrefix(key, keyPrefix)
	return strings.Split(key, ":")
}

func (dh *DataHandler) ShutDown() {
	close(dh.quit)
	dh.quitWG.Wait()
}

func zoneLocationsKey(zone string) string {
	return keyPrefix + zone + ":labels"
}

func zoneConfigKey(zone string) string {
	return keyPrefix + zone + ":config"
}

func zoneLocationRRSetKey(zone string, label string, rtype string) string {
	return keyPrefix + zone + ":labels:" + label + ":" + rtype
}

func zonePubKey(zone string, keyType string) string {
	return keyPrefix + zone + ":" + keyType + ":pub"
}

func zonePrivKey(zone string, keyType string) string {
	return keyPrefix + zone + ":" + keyType + ":priv"
}

func zoneWildcard(zone string) string {
	return keyPrefix + zone + ":*"
}

func zoneLocationsWildcard(zone string) string {
	return keyPrefix + zone + ":labels:*"
}

func locationWildcard(zone string, label string) string {
	return keyPrefix + zone + ":labels:" + label + ":*"
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
	if err := dh.redis.SAdd(zoneLocationsKey(zone), "@"); err != nil {
		return err
	}
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
		locations, err := dh.redis.SMembers(zoneLocationsKey(zone))
		if err != nil {
			zap.L().Error("cannot load zone locations", zap.String("zone", zone), zap.Error(err))
			return nil, err
		}

		configStr, err := dh.redis.Get(zoneConfigKey(zone))
		if err != nil {
			zap.L().Error("cannot load zone config", zap.String("zone", zone), zap.Error(err))
		}

		z := types.NewZone(zone, locations, configStr)
		dh.loadZoneKeys(z)
		z.CacheTimeout = time.Now().Unix() + dh.config.ZoneCacheTimeout

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

func (dh *DataHandler) EnableLocation(zone string, location string) error {
	return dh.redis.SAdd(zoneLocationsKey(zone), location)
}

func (dh *DataHandler) DisableLocation(zone string, location string) error {
	return dh.redis.SRem(zoneLocationsKey(zone), location)
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

func (dh *DataHandler) getRRSet(zone string, label string, rtype uint16, result types.RRSet) (types.RRSet, error) {
	key := zoneLocationRRSetKey(zone, label, types.TypeToString(rtype))
	cachedRRSet, found := dh.recordCache.Get(key)
	var r types.RRSet
	if found {
		r = cachedRRSet.(types.RRSet)
		if time.Now().Unix() <= dh.config.RecordCacheTimeout {
			return r, nil
		}
	}
	answer, err, _ := dh.recordInflight.Do(key, func() (interface{}, error) {
		val, err := dh.redis.Get(key)
		if err == redisCon.ErrNil {
			dh.recordCache.Set(key, result, 1)
			return result, nil
		} else if err != nil {
			zap.L().Error("cannot get location", zap.Error(err), zap.String("label", label), zap.String("zone", zone))
			return nil, err
		}
		if val != "" {
			err := jsoniter.Unmarshal([]byte(val), result)
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
		dh.recordCache.Set(key, result, 1)
		return result, nil
	})

	if answer == nil {
		return r, err
	}

	return answer.(types.RRSet), nil
}

func (dh *DataHandler) SetRRSetFromJson(zone string, label string, rtype uint16, value string) error {
	return dh.redis.Set(zoneLocationRRSetKey(zone, label, types.TypeToString(rtype)), value)
}

func (dh *DataHandler) SetRRSet(zone string, label string, rtype uint16, rrset types.RRSet) error {
	jsonValue, err := jsoniter.Marshal(rrset)
	if err != nil {
		return err
	}
	return dh.SetRRSetFromJson(zone, label, rtype, string(jsonValue))
}

type locationEntry struct {
	A     *types.IP_RRSet    `json:"a,omitempty"`
	AAAA  *types.IP_RRSet    `json:"aaaa,omitempty"`
	CNAME *types.CNAME_RRSet `json:"cname,omitempty"`
	TXT   *types.TXT_RRSet   `json:"txt,omitempty"`
	NS    *types.NS_RRSet    `json:"ns,omitempty"`
	MX    *types.MX_RRSet    `json:"mx,omitempty"`
	SRV   *types.SRV_RRSet   `json:"srv,omitempty"`
	CAA   *types.CAA_RRSet   `json:"caa,omitempty"`
	PTR   *types.PTR_RRSet   `json:"ptr,omitempty"`
	TLSA  *types.TLSA_RRSet  `json:"tlsa,omitempty"`
	DS    *types.DS_RRSet    `json:"ds,omitempty"`
	ANAME *types.ANAME_RRSet `json:"aname,omitempty"`
}

func (dh *DataHandler) SetLocationFromJson(zone string, location string, value string) error {
	var entry locationEntry
	if err := jsoniter.Unmarshal([]byte(value), &entry); err != nil {
		return err
	}
	if err := dh.EnableLocation(zone, location); err != nil {
		return err
	}
	if entry.A != nil {
		entry.A.TtlValue = fixTTL(entry.A.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeA, entry.A); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.AAAA != nil {
		entry.AAAA.TtlValue = fixTTL(entry.AAAA.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeAAAA, entry.AAAA); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.CNAME != nil {
		entry.CNAME.TtlValue = fixTTL(entry.CNAME.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeCNAME, entry.CNAME); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.TXT != nil {
		entry.TXT.TtlValue = fixTTL(entry.TXT.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeTXT, entry.TXT); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.NS != nil {
		entry.NS.TtlValue = fixTTL(entry.NS.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeNS, entry.NS); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.MX != nil {
		entry.MX.TtlValue = fixTTL(entry.MX.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeMX, entry.MX); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.SRV != nil {
		entry.SRV.TtlValue = fixTTL(entry.SRV.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeSRV, entry.SRV); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.CAA != nil {
		entry.CAA.TtlValue = fixTTL(entry.CAA.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeCAA, entry.CAA); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.PTR != nil {
		entry.PTR.TtlValue = fixTTL(entry.PTR.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypePTR, entry.PTR); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.TLSA != nil {
		entry.TLSA.TtlValue = fixTTL(entry.TLSA.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeTLSA, entry.TLSA); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.DS != nil {
		entry.DS.TtlValue = fixTTL(entry.DS.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, dns.TypeDS, entry.DS); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	if entry.ANAME != nil {
		entry.ANAME.TtlValue = fixTTL(entry.ANAME.TtlValue, dh.config.MinTTL, dh.config.MaxTTL)
		if err := dh.SetRRSet(zone, location, types.TypeANAME, entry.ANAME); err != nil {
			zap.L().Error("cannot set rrset", zap.Error(err))
		}
	}
	return nil
}

func fixTTL(ttl uint32, min uint32, max uint32) uint32 {
	if ttl < min {
		ttl = min
	}
	if ttl > max {
		ttl = max
	}
	return ttl
}

func (dh *DataHandler) A(zone string, label string) (*types.IP_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeA, &types.IP_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.IP_RRSet), nil
}

func (dh *DataHandler) AAAA(zone string, label string) (*types.IP_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeAAAA, &types.IP_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.IP_RRSet), nil
}

func (dh *DataHandler) CNAME(zone string, label string) (*types.CNAME_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeCNAME, &types.CNAME_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.CNAME_RRSet), nil
}

func (dh *DataHandler) TXT(zone string, label string) (*types.TXT_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeTXT, &types.TXT_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.TXT_RRSet), nil
}

func (dh *DataHandler) NS(zone string, label string) (*types.NS_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeNS, &types.NS_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.NS_RRSet), nil
}

func (dh *DataHandler) MX(zone string, label string) (*types.MX_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeMX, &types.MX_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.MX_RRSet), nil
}

func (dh *DataHandler) SRV(zone string, label string) (*types.SRV_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeSRV, &types.SRV_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.SRV_RRSet), nil
}

func (dh *DataHandler) CAA(zone string, label string) (*types.CAA_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeCAA, &types.CAA_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.CAA_RRSet), nil
}

func (dh *DataHandler) PTR(zone string, label string) (*types.PTR_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypePTR, &types.PTR_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.PTR_RRSet), nil
}

func (dh *DataHandler) TLSA(zone string, label string) (*types.TLSA_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeTLSA, &types.TLSA_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.TLSA_RRSet), nil
}

func (dh *DataHandler) DS(zone string, label string) (*types.DS_RRSet, error) {
	r, err := dh.getRRSet(zone, label, dns.TypeDS, &types.DS_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.DS_RRSet), nil
}

func (dh *DataHandler) ANAME(zone string, label string) (*types.ANAME_RRSet, error) {
	r, err := dh.getRRSet(zone, label, types.TypeANAME, &types.ANAME_RRSet{})
	if err != nil {
		return nil, err
	}
	return r.(*types.ANAME_RRSet), nil
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

func (dh *DataHandler) GetRevision() (int, error) {
	r, err := dh.redis.Get(revisionKey)
	if err != nil {
		if err == redisCon.ErrNil {
			return 0, nil
		}
		return 0, err
	}
	return strconv.Atoi(r)
}

func (dh *DataHandler) SetRevision(revision int) error {
	return dh.redis.Set(revisionKey, strconv.Itoa(revision))
}

func (dh *DataHandler) ApplyEvent(event database.Event) error {
	var tx hiredis.Transaction
	switch event.Type {
	case database.AddZone:
		var newZone database.NewZone
		if err := jsoniter.Unmarshal([]byte(event.Value), &newZone); err != nil {
			return err
		}
		config := &types.ZoneConfig{
			DomainId:        event.ZoneId,
			SOA:             &newZone.SOA,
			DnsSec:          newZone.Dnssec,
			CnameFlattening: newZone.CNameFlattening,
		}
		configJson, err := jsoniter.Marshal(config)
		if err != nil {
			return err
		}
		nsValue, err := jsoniter.Marshal(newZone.NS)
		if err != nil {
			return err
		}
		tx = dh.redis.Start()
		if newZone.Enabled {
			tx.SAdd(zonesKey, newZone.Name)
		}
		tx.
			Del(zoneWildcard(newZone.Name)).
			SAdd(zoneLocationsKey(newZone.Name), "@").
			Set(zoneConfigKey(newZone.Name), string(configJson)).
			Set(zoneLocationRRSetKey(newZone.Name, "@", types.TypeToString(dns.TypeNS)), string(nsValue)).
			Set(zonePubKey(newZone.Name, "ksk"), newZone.Keys.KSKPublic).
			Set(zonePrivKey(newZone.Name, "ksk"), newZone.Keys.KSKPrivate).
			Set(zonePubKey(newZone.Name, "zsk"), newZone.Keys.ZSKPublic).
			Set(zonePrivKey(newZone.Name, "zsk"), newZone.Keys.ZSKPrivate)
	case database.ImportZone:
		var importZone database.ZoneImport
		zap.L().Error("importZone", zap.String("data", event.Value))
		if err := jsoniter.Unmarshal([]byte(event.Value), &importZone); err != nil {
			return err
		}
		tx = dh.redis.Start()
		tx.
			Del(zoneLocationsWildcard(importZone.Name))
		for location, entry := range importZone.Entries {
			tx.SAdd(zoneLocationsKey(importZone.Name), location)
			for recordType, recordValue := range entry {
				value, _ := jsoniter.Marshal(recordValue)
				tx.Set(zoneLocationRRSetKey(importZone.Name, location, recordType), string(value))
			}
		}
	case database.UpdateZone:
		var zoneUpdate database.ZoneUpdate
		if err := jsoniter.Unmarshal([]byte(event.Value), &zoneUpdate); err != nil {
			return err
		}
		config := &types.ZoneConfig{
			DomainId:        event.ZoneId,
			SOA:             &zoneUpdate.SOA,
			DnsSec:          zoneUpdate.Dnssec,
			CnameFlattening: zoneUpdate.CNameFlattening,
		}
		configJson, err := jsoniter.Marshal(config)
		if err != nil {
			return err
		}
		tx = dh.redis.Start()
		if zoneUpdate.Enabled {
			tx.SAdd(zonesKey, zoneUpdate.Name)
		} else {
			tx.SRem(zonesKey, zoneUpdate.Name)
		}
		tx.
			SAdd(zoneLocationsKey(zoneUpdate.Name), "@").
			Set(zoneConfigKey(zoneUpdate.Name), string(configJson))
	case database.DeleteZone:
		var zoneDelete database.ZoneDelete
		if err := jsoniter.Unmarshal([]byte(event.Value), &zoneDelete); err != nil {
			return err
		}
		tx = dh.redis.Start()
		tx.
			SRem(zonesKey, zoneDelete.Name).
			Del(zoneWildcard(zoneDelete.Name))
	case database.AddLocation:
		var newLocation database.NewLocation
		if err := jsoniter.Unmarshal([]byte(event.Value), &newLocation); err != nil {
			return err
		}
		tx = dh.redis.Start()
		if newLocation.Enabled {
			tx.SAdd(zoneLocationsKey(newLocation.ZoneName), newLocation.Location)
		}
	case database.UpdateLocation:
		var locationUpdate database.LocationUpdate
		if err := jsoniter.Unmarshal([]byte(event.Value), &locationUpdate); err != nil {
			return err
		}
		tx = dh.redis.Start()
		if locationUpdate.Enabled {
			tx.SAdd(zoneLocationsKey(locationUpdate.ZoneName), locationUpdate.Location)
		} else {
			tx.SRem(zoneLocationsKey(locationUpdate.ZoneName), locationUpdate.Location)
		}
	case database.DeleteLocation:
		var locationDelete database.LocationDelete
		if err := jsoniter.Unmarshal([]byte(event.Value), &locationDelete); err != nil {
			return err
		}
		tx = dh.redis.Start()
		tx.
			SRem(zoneLocationsKey(locationDelete.ZoneName), locationDelete.Location).
			Del(locationWildcard(locationDelete.ZoneName, locationDelete.Location))
	case database.AddRecord:
		var newRecord database.NewRecordSet
		if err := jsoniter.Unmarshal([]byte(event.Value), &newRecord); err != nil {
			return err
		}
		tx = dh.redis.Start()
		if newRecord.Enabled {
			value, _ := jsoniter.Marshal(newRecord.Value)
			tx.Set(zoneLocationRRSetKey(newRecord.ZoneName, newRecord.Location, newRecord.Type), string(value))
		}
	case database.UpdateRecord:
		var recordUpdate database.RecordSetUpdate
		if err := jsoniter.Unmarshal([]byte(event.Value), &recordUpdate); err != nil {
			return err
		}
		tx = dh.redis.Start()
		if recordUpdate.Enabled {
			value, _ := jsoniter.Marshal(recordUpdate.Value)
			tx.Set(zoneLocationRRSetKey(recordUpdate.ZoneName, recordUpdate.Location, recordUpdate.Type), string(value))
		} else {
			tx.Del(zoneLocationRRSetKey(recordUpdate.ZoneName, recordUpdate.Location, recordUpdate.Type))
		}
	case database.DeleteRecord:
		var recordDelete database.RecordSetDelete
		if err := jsoniter.Unmarshal([]byte(event.Value), &recordDelete); err != nil {
			return err
		}
		tx = dh.redis.Start()
		tx.Del(zoneLocationRRSetKey(recordDelete.ZoneName, recordDelete.Location, recordDelete.Type))
	default:
		return fmt.Errorf("invalid event type: %s", event.Type)
	}
	return tx.Set(revisionKey, strconv.Itoa(event.Revision)).Commit()
}

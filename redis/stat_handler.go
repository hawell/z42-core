package redis

import (
	"github.com/dgraph-io/ristretto"
	"github.com/hawell/logger"
	"github.com/hawell/z42/types"
	jsoniter "github.com/json-iterator/go"
	"strings"
	"time"
)

const (
	cacheSize = 100000
)

type StatHandlerConfig struct {
	Redis RedisConfig `json:"redis"`
}

type StatHandler struct {
	Redis *Redis
	cache *ristretto.Cache
}

func NewStatHandler(config *StatHandlerConfig) *StatHandler {
	result := &StatHandler{
		Redis: NewRedis(&config.Redis),
	}
	result.cache, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(cacheSize) * 10,
		MaxCost:     int64(cacheSize),
		BufferItems: 64,
		Metrics:     false,
	})
	return result
}

func (sh *StatHandler) GetActiveHealthcheckItems() ([]string, error) {
	itemKeys, err := sh.Redis.GetKeys("z42:healthcheck:*")
	if err != nil {
		return nil, err
	}
	var res []string
	for i := range itemKeys {
		res = append(res, strings.TrimPrefix(itemKeys[i], "z42:healthcheck:"))
	}
	return res, nil
}

func (sh *StatHandler) GetHealthcheckItem(key string) (*types.HealthCheckItem, error) {
	item := new(types.HealthCheckItem)
	itemStr, err := sh.Redis.Get("z42:healthcheck:" + key)
	if err != nil {
		logger.Default.Errorf("cannot load item %s : %s", key, err)
		return nil, err
	}
	jsoniter.Unmarshal([]byte(itemStr), item)
	if item.DownCount > 0 {
		item.DownCount = -item.DownCount
	}
	return item, nil
}

func (sh *StatHandler) SetHealthcheckItem(item *types.HealthCheckItem) error {
	key := item.Host + ":" + item.Ip
	itemStr, err := jsoniter.Marshal(item)
	if err != nil {
		logger.Default.Errorf("cannot marshal item to json : %s", err)
		return err
	}
	// logger.Default.Debugf("setting %v in redis : %s", *item, string(itemStr))
	sh.Redis.Set("z42:healthcheck:"+key, string(itemStr))
	return nil
}

func (sh *StatHandler) SetHealthcheckItemExpiration(key string, lifespan time.Duration) error {
	return sh.Redis.Expire("z42:healthcheck:"+key, lifespan)
}

func (sh *StatHandler) GetHealthStatus(domain string, ip string) int {
	key := domain + ":" + ip
	var (
		item *types.HealthCheckItem
		err  error
	)
	val, found := sh.cache.Get(key)
	if !found {
		item, err = sh.GetHealthcheckItem(key)
		if err != nil {
			return 0
		}
		if item == nil {
			item = new(types.HealthCheckItem)
		}
		sh.cache.Set(key, item, 1)
	} else {
		item = val.(*types.HealthCheckItem)
	}
	return item.Status
}

func (sh *StatHandler) ShutDown() {

}

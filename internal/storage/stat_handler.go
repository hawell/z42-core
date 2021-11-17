package storage

import (
	"github.com/dgraph-io/ristretto"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/pkg/hiredis"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

const (
	cacheSize = 100000
)

type StatHandler struct {
	redis  *hiredis.Redis
	cache  *ristretto.Cache
	quit   chan struct{}
	quitWG sync.WaitGroup
}

func NewStatHandler(config *StatHandlerConfig) *StatHandler {
	sh := &StatHandler{
		redis: hiredis.NewRedis(&config.Redis),
		quit:  make(chan struct{}),
	}
	sh.cache, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(cacheSize) * 10,
		MaxCost:     int64(cacheSize),
		BufferItems: 64,
		Metrics:     false,
	})

	go func() {
		sh.quitWG.Add(1)
		quit := make(chan *sync.WaitGroup, 1)
		go sh.redis.SubscribeEvent("z42:healthcheck:*",
			func() {
			},
			func(channel string, data string) {
				key := strings.TrimLeft(channel, "z42:healthcheck:")
				sh.cache.Del(key)
			},
			func(err error) {
				zap.L().Error("subscribe error", zap.Error(err))
			},
			quit)

		<-sh.quit
		quit <- &sh.quitWG
	}()

	return sh
}

func (sh *StatHandler) GetActiveHealthcheckItems() ([]string, error) {
	itemKeys, err := sh.redis.GetKeys("z42:healthcheck:*")
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
	itemStr, err := sh.redis.Get("z42:healthcheck:" + key)
	if err != nil {
		zap.L().Error("cannot load item", zap.String("key", key), zap.Error(err))
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
		zap.L().Error("cannot marshal item to json", zap.Error(err))
		return err
	}
	sh.redis.Set("z42:healthcheck:"+key, string(itemStr))
	return nil
}

func (sh *StatHandler) SetHealthcheckItemExpiration(key string, lifespan time.Duration) error {
	return sh.redis.Expire("z42:healthcheck:"+key, lifespan)
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
	close(sh.quit)
	sh.quitWG.Wait()
}

func (sh StatHandler) Clear() error {
	return sh.redis.Del("*")
}

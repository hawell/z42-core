package redis

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	redisCon "github.com/gomodule/redigo/redis"
)

type Redis struct {
	Config *RedisConfig
	Pool   *redisCon.Pool
}

type RedisConnectionConfig struct {
	MaxIdleConnections   int  `json:"max_idle_connections"`
	MaxActiveConnections int  `json:"max_active_connections"`
	ConnectTimeout       int  `json:"connect_timeout"`
	ReadTimeout          int  `json:"read_timeout"`
	IdleKeepAlive        int  `json:"idle_keep_alive"`
	MaxKeepAlive         int  `json:"max_keep_alive"`
	WaitForConnection    bool `json:"wait_for_connection"`
}

type RedisConfig struct {
	Address    string                `json:"address"`
	Net        string                `json:"net"`
	DB         int                   `json:"db"`
	Password   string                `json:"password"`
	Prefix     string                `json:"prefix"`
	Suffix     string                `json:"suffix"`
	Connection RedisConnectionConfig `json:"connection"`
}

var noConnectionError = errors.New("no connection")

func NewRedis(config *RedisConfig) *Redis {
	r := &Redis{
		Config: config,
	}

	r.Pool = &redisCon.Pool{
		Dial: func() (redisCon.Conn, error) {
			var opts []redisCon.DialOption
			if r.Config.Password != "" {
				opts = append(opts, redisCon.DialPassword(r.Config.Password))
			}
			if r.Config.Connection.ConnectTimeout != 0 {
				opts = append(opts, redisCon.DialConnectTimeout(time.Duration(r.Config.Connection.ConnectTimeout)*time.Millisecond))
			}
			if r.Config.Connection.ReadTimeout != 0 {
				opts = append(opts, redisCon.DialReadTimeout(time.Duration(r.Config.Connection.ReadTimeout)*time.Millisecond))
			}
			opts = append(opts, redisCon.DialDatabase(r.Config.DB))

			return redisCon.Dial(r.Config.Net, r.Config.Address, opts...)
		},
		TestOnBorrow: func(c redisCon.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:         config.Connection.MaxIdleConnections,
		MaxActive:       config.Connection.MaxActiveConnections,
		IdleTimeout:     time.Second * time.Duration(config.Connection.IdleKeepAlive),
		Wait:            config.Connection.WaitForConnection,
		MaxConnLifetime: time.Second * time.Duration(config.Connection.MaxKeepAlive),
	}

	return r
}

func (redis *Redis) GetConfig(config string) (string, error) {
	var (
		err   error
		reply interface{}
		vals  []string
	)
	conn := redis.Pool.Get()
	if conn == nil {
		return "", noConnectionError
	}
	defer conn.Close()

	reply, err = conn.Do("CONFIG", "GET", config)
	if err != nil {
		return "", err
	}
	vals, err = redisCon.Strings(reply, nil)
	if err != nil {
		return "", err
	}
	return vals[1], nil
}

func (redis *Redis) SetConfig(config string, value string) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	_, err := conn.Do("CONFIG", "SET", config, value)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) Get(key string) (string, error) {
	var (
		err   error
		reply interface{}
		val   string
	)
	conn := redis.Pool.Get()
	if conn == nil {
		return "", noConnectionError
	}
	defer conn.Close()

	reply, err = conn.Do("GET", redis.Config.Prefix+key+redis.Config.Suffix)
	if err != nil {
		return "", err
	}
	val, err = redisCon.String(reply, nil)
	if err != nil {
		return "", err
	}
	return val, nil
}

func (redis *Redis) Set(key string, value string) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	_, err := conn.Do("SET", redis.Config.Prefix+key+redis.Config.Suffix, value)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) Del(pattern string) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	keys, err := redis.GetKeys(pattern)
	if err != nil {
		return err
	}
	if keys == nil || len(keys) == 0 {
		return nil
	}
	var arg []interface{}
	for i := range keys {
		arg = append(arg, redis.Config.Prefix+keys[i]+redis.Config.Suffix)
	}
	_, err = conn.Do("DEL", arg...)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) GetKeys(pattern string) ([]string, error) {
	var (
		reply interface{}
		err   error
		keys  []string
	)

	conn := redis.Pool.Get()
	if conn == nil {
		return nil, noConnectionError
	}
	defer conn.Close()

	keySet := make(map[string]interface{})

	cursor := "0"
	for {
		reply, err = conn.Do("SCAN", cursor, "MATCH", redis.Config.Prefix+pattern+redis.Config.Suffix, "COUNT", 100)
		if err != nil {
			return nil, err
		}
		var values []interface{}
		values, err = redisCon.Values(reply, nil)
		if err != nil {
			return nil, err
		}
		cursor, err = redisCon.String(values[0], nil)
		if err != nil {
			return nil, err
		}
		keys, err = redisCon.Strings(values[1], nil)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			keySet[key] = nil
		}
		if cursor == "0" {
			break
		}
	}
	keys = []string{}
	for key := range keySet {
		key = strings.TrimPrefix(key, redis.Config.Prefix)
		key = strings.TrimSuffix(key, redis.Config.Suffix)
		keys = append(keys, key)
	}
	return keys, nil
}

func (redis *Redis) GetHKeys(key string) ([]string, error) {
	var (
		reply   interface{}
		err     error
		keyvals map[string]string
	)

	conn := redis.Pool.Get()
	if conn == nil {
		return nil, noConnectionError
	}
	defer conn.Close()

	keySet := make(map[string]interface{})

	cursor := "0"
	for {
		reply, err = conn.Do("HSCAN", redis.Config.Prefix+key+redis.Config.Suffix, cursor, "COUNT", 100)
		if err != nil {
			return nil, err
		}
		var values []interface{}
		values, err = redisCon.Values(reply, nil)
		if err != nil {
			return nil, err
		}
		cursor, err = redisCon.String(values[0], nil)
		if err != nil {
			return nil, err
		}
		keyvals, err = redisCon.StringMap(values[1], nil)
		if err != nil {
			return nil, err
		}
		for key := range keyvals {
			keySet[key] = nil
		}
		if cursor == "0" {
			break
		}
	}
	keys := make([]string, len(keySet))
	i := 0
	for key := range keySet {
		keys[i] = key
		i++
	}
	return keys, nil
}

func (redis *Redis) HGet(key string, hkey string) (string, error) {
	var (
		err   error
		reply interface{}
		val   string
	)
	conn := redis.Pool.Get()
	if conn == nil {
		return "", noConnectionError
	}
	defer conn.Close()

	reply, err = conn.Do("HGET", redis.Config.Prefix+key+redis.Config.Suffix, hkey)
	if err != nil {
		return "", err
	}
	val, err = redisCon.String(reply, nil)
	if err != nil {
		return "", err
	}
	return val, nil
}

func (redis *Redis) HSet(key string, hkey string, value string) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	// log.Printf("[DEBUG] HSET : %s %s %s", redis.config.prefix + key + redis.config.suffix, hkey, value)
	_, err := conn.Do("HSET", redis.Config.Prefix+key+redis.Config.Suffix, hkey, value)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) SAdd(set string, member string) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	_, err := conn.Do("SADD", redis.Config.Prefix+set+redis.Config.Suffix, member)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) SRem(set string, member string) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	_, err := conn.Do("SREM", redis.Config.Prefix+set+redis.Config.Suffix, member)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) SIsMember(set string, member string) (bool, error) {
	conn := redis.Pool.Get()
	if conn == nil {
		return false, noConnectionError
	}
	defer conn.Close()

	reply, err := conn.Do("SISMEMBER", redis.Config.Prefix+set+redis.Config.Suffix, member)
	if err != nil {
		return false, err
	}
	val, err := redisCon.Bool(reply, nil)
	if err != nil {
		return false, err
	}
	return val, nil
}

func (redis *Redis) SMembers(set string) ([]string, error) {
	var (
		reply interface{}
		err   error
		keys  []string
	)

	conn := redis.Pool.Get()
	if conn == nil {
		return nil, noConnectionError
	}
	defer conn.Close()

	keySet := make(map[string]interface{})

	cursor := "0"
	for {
		reply, err = conn.Do("SSCAN", redis.Config.Prefix+set+redis.Config.Suffix, cursor, "COUNT", 100)
		if err != nil {
			return nil, err
		}
		var values []interface{}
		values, err = redisCon.Values(reply, nil)
		if err != nil {
			return nil, err
		}
		cursor, err = redisCon.String(values[0], nil)
		if err != nil {
			return nil, err
		}
		keys, err = redisCon.Strings(values[1], nil)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			keySet[key] = nil
		}
		if cursor == "0" {
			break
		}
	}
	keys = []string{}
	for key := range keySet {
		keys = append(keys, key)
	}
	return keys, nil
}

type MessageHandler func(channel string, event string)

func (redis *Redis) SubscribeEvent(pattern string, onStart func(), onMessage func(channel string, data string), onError func(err error), quit chan *sync.WaitGroup) {
	done := make(chan error, 1)
	var psc *redisCon.PubSubConn = nil
	channelPrefix := "__keyspace@" + strconv.Itoa(redis.Config.DB) + "__:"
	Init := func() error {
		conn := redis.Pool.Get()
		if conn == nil {
			return errors.New("no connection")
		}

		newPsc := &redisCon.PubSubConn{Conn: conn}
		if err := newPsc.PSubscribe(channelPrefix + redis.Config.Prefix + pattern + redis.Config.Suffix); err != nil {
			newPsc.Close()
			return err
		}
		psc = newPsc
		return nil
	}
	Subscribe := func() {
		onStart()
		defer psc.Close()
		for {
			switch n := psc.ReceiveWithTimeout(time.Minute * 2).(type) {
			case error:
				done <- n
				return
			case redisCon.Message:
				channel := strings.TrimPrefix(n.Channel, channelPrefix+redis.Config.Prefix)
				channel = strings.TrimSuffix(channel, redis.Config.Suffix)
				onMessage(channel, string(n.Data))
			case redisCon.Subscription:
				if n.Kind == "unsubscribe" || n.Kind == "punsubscribe" {
					done <- nil
					return
				}
			default:
			}
		}
	}

	if err := Init(); err != nil {
		onError(err)
	} else {
		go Subscribe()
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if psc == nil {
				if err := Init(); err != nil {
					onError(err)
				} else {
					go Subscribe()
				}
			} else {
				psc.Ping("")
			}
		case wg := <-quit:
			if psc != nil {
				psc.PUnsubscribe(channelPrefix + redis.Config.Prefix + pattern + redis.Config.Suffix)
			}
			<-done
			wg.Done()
			return
		case err := <-done:
			if err != nil {
				onError(err)
				psc = nil
			}
		}
	}
}

func (redis *Redis) Expire(key string, duration time.Duration) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()
	_, err := conn.Do("PEXPIRE", redis.Config.Prefix+key+redis.Config.Suffix, duration.Nanoseconds()/1000000)
	return err
}

func (redis *Redis) Persist(key string) error {
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()
	_, err := conn.Do("PERSIST", redis.Config.Prefix+key+redis.Config.Suffix)
	return err
}

func (redis *Redis) Ping() error {
	var (
		err   error
		reply interface{}
		val   string
	)
	conn := redis.Pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	reply, err = conn.Do("PING")
	if err != nil {
		return err
	}
	val, err = redisCon.String(reply, nil)
	if err != nil {
		return err
	}
	if val != "PONG" {
		return errors.New("PING failed")
	}
	return nil
}

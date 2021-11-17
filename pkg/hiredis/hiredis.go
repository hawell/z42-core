package hiredis

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	redisCon "github.com/gomodule/redigo/redis"
)

type Redis struct {
	config *Config
	pool   *redisCon.Pool
}

var noConnectionError = errors.New("no connection")

func NewRedis(config *Config) *Redis {
	r := &Redis{
		config: config,
	}

	r.pool = &redisCon.Pool{
		Dial: func() (redisCon.Conn, error) {
			var opts []redisCon.DialOption
			if r.config.Password != "" {
				opts = append(opts, redisCon.DialPassword(r.config.Password))
			}
			if r.config.Connection.ConnectTimeout != 0 {
				opts = append(opts, redisCon.DialConnectTimeout(time.Duration(r.config.Connection.ConnectTimeout)*time.Millisecond))
			}
			if r.config.Connection.ReadTimeout != 0 {
				opts = append(opts, redisCon.DialReadTimeout(time.Duration(r.config.Connection.ReadTimeout)*time.Millisecond))
			}
			opts = append(opts, redisCon.DialDatabase(r.config.DB))

			return redisCon.Dial(r.config.Net, r.config.Address, opts...)
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
	conn := redis.pool.Get()
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
	conn := redis.pool.Get()
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
	conn := redis.pool.Get()
	if conn == nil {
		return "", noConnectionError
	}
	defer conn.Close()

	reply, err = conn.Do("GET", redis.config.Prefix+key+redis.config.Suffix)
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
	conn := redis.pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	_, err := conn.Do("SET", redis.config.Prefix+key+redis.config.Suffix, value)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) Del(pattern string) error {
	conn := redis.pool.Get()
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
		arg = append(arg, redis.config.Prefix+keys[i]+redis.config.Suffix)
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

	conn := redis.pool.Get()
	if conn == nil {
		return nil, noConnectionError
	}
	defer conn.Close()

	keySet := make(map[string]interface{})

	cursor := "0"
	for {
		reply, err = conn.Do("SCAN", cursor, "MATCH", redis.config.Prefix+pattern+redis.config.Suffix, "COUNT", 100)
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
		key = strings.TrimPrefix(key, redis.config.Prefix)
		key = strings.TrimSuffix(key, redis.config.Suffix)
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

	conn := redis.pool.Get()
	if conn == nil {
		return nil, noConnectionError
	}
	defer conn.Close()

	keySet := make(map[string]interface{})

	cursor := "0"
	for {
		reply, err = conn.Do("HSCAN", redis.config.Prefix+key+redis.config.Suffix, cursor, "COUNT", 100)
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
	conn := redis.pool.Get()
	if conn == nil {
		return "", noConnectionError
	}
	defer conn.Close()

	reply, err = conn.Do("HGET", redis.config.Prefix+key+redis.config.Suffix, hkey)
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
	conn := redis.pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	// log.Printf("[DEBUG] HSET : %s %s %s", redis.config.prefix + key + redis.config.suffix, hkey, value)
	_, err := conn.Do("HSET", redis.config.Prefix+key+redis.config.Suffix, hkey, value)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) SAdd(set string, member string) error {
	conn := redis.pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	_, err := conn.Do("SADD", redis.config.Prefix+set+redis.config.Suffix, member)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) SRem(set string, member string) error {
	conn := redis.pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()

	_, err := conn.Do("SREM", redis.config.Prefix+set+redis.config.Suffix, member)
	if err != nil {
		return err
	}
	return nil
}

func (redis *Redis) SIsMember(set string, member string) (bool, error) {
	conn := redis.pool.Get()
	if conn == nil {
		return false, noConnectionError
	}
	defer conn.Close()

	reply, err := conn.Do("SISMEMBER", redis.config.Prefix+set+redis.config.Suffix, member)
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

	conn := redis.pool.Get()
	if conn == nil {
		return nil, noConnectionError
	}
	defer conn.Close()

	keySet := make(map[string]interface{})

	cursor := "0"
	for {
		reply, err = conn.Do("SSCAN", redis.config.Prefix+set+redis.config.Suffix, cursor, "COUNT", 100)
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
	channelPrefix := "__keyspace@" + strconv.Itoa(redis.config.DB) + "__:"
	Init := func() error {
		conn := redis.pool.Get()
		if conn == nil {
			return errors.New("no connection")
		}

		newPsc := &redisCon.PubSubConn{Conn: conn}
		if err := newPsc.PSubscribe(channelPrefix + redis.config.Prefix + pattern + redis.config.Suffix); err != nil {
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
				channel := strings.TrimPrefix(n.Channel, channelPrefix+redis.config.Prefix)
				channel = strings.TrimSuffix(channel, redis.config.Suffix)
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
				psc.PUnsubscribe(channelPrefix + redis.config.Prefix + pattern + redis.config.Suffix)
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
	conn := redis.pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()
	_, err := conn.Do("PEXPIRE", redis.config.Prefix+key+redis.config.Suffix, duration.Nanoseconds()/1000000)
	return err
}

func (redis *Redis) Persist(key string) error {
	conn := redis.pool.Get()
	if conn == nil {
		return noConnectionError
	}
	defer conn.Close()
	_, err := conn.Do("PERSIST", redis.config.Prefix+key+redis.config.Suffix)
	return err
}

func (redis *Redis) Ping() error {
	var (
		err   error
		reply interface{}
		val   string
	)
	conn := redis.pool.Get()
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

type StreamItem struct {
	ID    string
	Key   string
	Value string
}

func (redis *Redis) XAdd(stream string, kv StreamItem) (string, error) {
	conn := redis.pool.Get()
	if conn == nil {
		return "", noConnectionError
	}
	defer conn.Close()

	if kv.ID == "" {
		kv.ID = "*"
	}
	reply, err := conn.Do("XADD", redis.config.Prefix+stream+redis.config.Suffix, kv.ID, kv.Key, kv.Value)
	if err != nil {
		return "", err
	}
	val, err := redisCon.String(reply, nil)
	if err != nil {
		return "", err
	}
	return val, nil
}

func (redis *Redis) XRead(streamID string, lastID string) ([]StreamItem, error) {
	conn := redis.pool.Get()
	if conn == nil {
		return nil, noConnectionError
	}
	defer conn.Close()

	if lastID == "" {
		lastID = "$"
	}

	reply, err := conn.Do("XREAD", "BLOCK", "0", "STREAMS", redis.config.Prefix+streamID+redis.config.Suffix, lastID)
	if err != nil {
		return nil, err
	}
	streams, err := redisCon.Values(reply, nil)
	if err != nil {
		return nil, err
	}
	stream, err := redisCon.Values(streams[0], nil)
	if err != nil {
		return nil, err
	}
	var res []StreamItem
	values, err := redisCon.Values(stream[1], nil)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		item, err := redisCon.Values(value, nil)
		if err != nil {
			return nil, err
		}
		id, err := redisCon.String(item[0], nil)
		if err != nil {
			return nil, err
		}
		kv, err := redisCon.Strings(item[1], nil)
		if err != nil {
			return nil, err
		}
		res = append(res, StreamItem{id, kv[0], kv[1]})
	}
	return res, nil
}

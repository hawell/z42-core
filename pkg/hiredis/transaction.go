package hiredis

import redisCon "github.com/gomodule/redigo/redis"

type Transaction struct {
	connection redisCon.Conn
	redis      *Redis
	error      error
}

func (redis *Redis) Start() Transaction {
	conn := redis.pool.Get()
	if conn == nil {
		return Transaction{
			error: noConnectionError,
		}
	}
	t := Transaction{
		connection: conn,
		redis:      redis,
		error:      nil,
	}
	t.error = t.connection.Send("MULTI")
	return t
}

func (t Transaction) Commit() error {
	defer func() {
		if t.connection != nil {
			t.connection.Close()
		}
	}()
	if t.error != nil {
		return t.error
	}
	_, err := t.connection.Do("EXEC")
	return err
}

func (t Transaction) Set(key string, value string) Transaction {
	if t.error != nil {
		return t
	}
	t.error = t.connection.Send("SET", t.redis.config.Prefix+key+t.redis.config.Suffix, value)
	return t
}

func (t Transaction) Del(key string) Transaction {
	if t.error != nil {
		return t
	}
	t.error = t.connection.Send("DEL", t.redis.config.Prefix+key+t.redis.config.Suffix)
	return t
}

func (t Transaction) HSet(key string, hkey string, value string) Transaction {
	if t.error != nil {
		return t
	}
	t.error = t.connection.Send("HSET", t.redis.config.Prefix+key+t.redis.config.Suffix, hkey, value)
	return t
}

func (t Transaction) SAdd(set string, member string) Transaction {
	if t.error != nil {
		return t
	}
	t.error = t.connection.Send("SADD", t.redis.config.Prefix+set+t.redis.config.Suffix, member)
	return t
}

func (t Transaction) SRem(set string, member string) Transaction {
	if t.error != nil {
		return t
	}
	t.error = t.connection.Send("SREM", t.redis.config.Prefix+set+t.redis.config.Suffix, member)
	return t
}

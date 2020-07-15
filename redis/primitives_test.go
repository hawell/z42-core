package redis

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/hawell/logger"
)

func TestRedis(t *testing.T) {
	cfg := RedisConfig{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("*")
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	err = r.Del("1")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	err = r.Set("1", "1")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	if v, _ := r.Get("1"); v != "1" {
		fmt.Println("1")
		t.Fail()
	}

	err = r.Del("2")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	err = r.HSet("2", "key1", "value1")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	err = r.HSet("2", "key2", "value2")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	hkeys, _ := r.GetHKeys("2")
	if !(hkeys[0] == "key1" && hkeys[1] == "key2") && !(hkeys[0] == "key2" && hkeys[1] == "key1") {
		fmt.Println("2")
		fmt.Println(hkeys[0], hkeys[1])
		t.Fail()
	}

	if v, _ := r.HGet("2", "key1"); v != "value1" {
		fmt.Println("3")
		t.Fail()
	}
	if v, _ := r.HGet("2", "key2"); v != "value2" {
		fmt.Println("4")
		t.Fail()
	}

	if v, _ := r.GetKeys("*"); len(v) != 2 {
		fmt.Println("5")
		t.Fail()
	}
	fmt.Println(r.GetKeys("*"))
	err = r.Del("*")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	fmt.Println(r.GetKeys("*"))
	if v, _ := r.GetKeys("*"); len(v) != 0 {
		fmt.Println("6")
		t.Fail()
	}
}

func TestConfig(t *testing.T) {
	var err error
	cfg := RedisConfig{
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)

	if err = r.SetConfig("notify-keyspace-events", "AE"); err != nil {
		fmt.Println("1 ", err)
		t.Fail()
	}
	config, err := r.GetConfig("notify-keyspace-events")
	if config != "AE" {
		fmt.Println(config, err)
		t.Fail()
	}
	if err := r.SetConfig("notify-keyspace-events", "AKE"); err != nil {
		fmt.Println("3 ", err)
		t.Fail()
	}
	config, _ = r.GetConfig("notify-keyspace-events")
	if config != "AKE" {
		fmt.Println(config, err)
		t.Fail()
	}
}

func TestPubSub(t *testing.T) {
	cfg := RedisConfig{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)

	r.Del("foo")
	handled := false
	wg := &sync.WaitGroup{}
	wg.Add(1)
	quit := make(chan *sync.WaitGroup, 1)
	go r.SubscribeEvent("foo", func() {
		fmt.Println("start")
	},
		func(channel string, data string) {
			handled = true
			if channel != "foo" {
				fmt.Println("1 ", channel)
				t.Fail()
			}
			if data != "set" {
				fmt.Println("2 ", data)
				t.Fail()
			}
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	r.Set("foo", "bar")
	time.Sleep(time.Millisecond * 200)
	if !handled {
		fmt.Println("3 event not handled")
		t.Fail()
	}
	quit <- wg
	wg.Wait()

	r.Del("faz")
	handled = false
	wg.Add(1)
	go r.SubscribeEvent("faz", func() {
		fmt.Println("start")
	},
		func(channel string, event string) {
			handled = true
			if channel != "faz" {
				fmt.Println("4 ", channel)
				t.Fail()
			}
			if event != "set" {
				fmt.Println("5 ", event)
				t.Fail()
			}
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	r.Set("faz", "bar")
	time.Sleep(time.Millisecond * 200)
	if !handled {
		fmt.Println("6 event not handled")
		t.Fail()
	}
	quit <- wg
	wg.Wait()

	r.Del("bar")
	handled = false
	wg.Add(1)
	go r.SubscribeEvent("bar", func() {
		fmt.Println("start")
	},
		func(channel string, event string) {
			handled = true
			if channel != "bar" {
				fmt.Println("7 ", channel)
				t.Fail()
			}
			if event != "hset" {
				fmt.Println("8 ", event)
				t.Fail()
			}
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	r.HSet("bar", "baz", "baa")
	time.Sleep(time.Millisecond * 200)
	if !handled {
		fmt.Println("9 event not handled")
		t.Fail()
	}
	quit <- wg
	wg.Wait()
}

func TestExpirePersist(t *testing.T) {
	cfg := RedisConfig{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	r.Del("foo")
	r.Set("foo", "bar")
	if v, _ := r.Get("foo"); v != "bar" {
		logger.Default.Error("1")
		t.Fail()
	}
	if r.Expire("foo", time.Millisecond*10) != nil {
		logger.Default.Error("2")
		t.Fail()
	}
	time.Sleep(time.Millisecond * 11)
	if v, _ := r.Get("foo"); v != "" {
		logger.Default.Error("3")
		t.Fail()
	}
	r.Set("foo", "bar")
	r.Expire("foo", time.Millisecond*10)
	time.Sleep(time.Millisecond)
	if v, _ := r.Get("foo"); v != "bar" {
		logger.Default.Error("4")
		t.Fail()
	}
	r.Persist("foo")
	time.Sleep(time.Millisecond * 11)
	if v, _ := r.Get("foo"); v != "bar" {
		logger.Default.Error("5")
		t.Fail()
	}
}

func TestSets(t *testing.T) {
	cfg := RedisConfig{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	r.Del("*")

	r.SAdd("set1", "value1")
	if v, _ := r.SIsMember("set1", "value1"); v != true {
		t.Fail()
	}
	r.SRem("set1", "value1")
	if v, _ := r.SIsMember("set1", "value1"); v != false {
		t.Fail()
	}

	r.SAdd("set1", "value1")
	r.SAdd("set1", "value2")
	r.SAdd("set1", "value3")
	r.SAdd("set1", "value4")
	members, _ := r.SMembers("set1")
	log.Println(members)
	if len(members) != 4 {
		t.Fail()
	}
}

func TestUnixSocket(t *testing.T) {
	cfg := RedisConfig{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "/var/run/redis/redis-server.sock",
		Net:     "unix",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Set("unix_test", "hello")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	v, err := r.Get("unix_test")
	if err != nil || v != "hello" {
		t.Fail()
	}
}

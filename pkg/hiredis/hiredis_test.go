package hiredis

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/gomega"
	"log"
	"sync"
	"testing"
	"time"
)

func TestRedis(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "127.0.0.1:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("*")
	Expect(err).To(BeNil())

	err = r.Del("1")
	Expect(err).To(BeNil())
	err = r.Set("1", "1")
	Expect(err).To(BeNil())
	v, err := r.Get("1")
	Expect(err).To(BeNil())
	Expect(v).To(Equal("1"))

	err = r.Del("2")
	Expect(err).To(BeNil())
	err = r.HSet("2", "key1", "value1")
	Expect(err).To(BeNil())
	err = r.HSet("2", "key2", "value2")
	Expect(err).To(BeNil())
	hkeys, err := r.GetHKeys("2")
	Expect(err).To(BeNil())
	Expect((hkeys[0] == "key1" && hkeys[1] == "key2") || (hkeys[0] == "key2" && hkeys[1] == "key1")).To(Equal(true))

	v, err = r.HGet("2", "key1")
	Expect(err).To(BeNil())
	Expect(v).To(Equal("value1"))
	v, err = r.HGet("2", "key2")
	Expect(err).To(BeNil())
	Expect(v).To(Equal("value2"))

	l, err := r.GetKeys("*")
	Expect(err).To(BeNil())
	Expect(len(l)).To(Equal(2))
	err = r.Del("*")
	Expect(err).To(BeNil())
	l, err = r.GetKeys("*")
	Expect(err).To(BeNil())
	Expect(len(l)).To(Equal(0))
}

func TestConfig(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Address: "127.0.0.1:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)

	err := r.SetConfig("notify-keyspace-events", "AE")
	Expect(err).To(BeNil())
	config, err := r.GetConfig("notify-keyspace-events")
	Expect(err).To(BeNil())
	Expect(config).To(Equal("AE"))
	err = r.SetConfig("notify-keyspace-events", "AKE")
	Expect(err).To(BeNil())
	config, err = r.GetConfig("notify-keyspace-events")
	Expect(err).To(BeNil())
	Expect(config).To(Equal("AKE"))
}

func TestPubSub(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "127.0.0.1:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)

	err := r.SetConfig("notify-keyspace-events", "AKE")
	Expect(err).To(BeNil())
	err = r.Del("foo")
	Expect(err).To(BeNil())
	handled := false
	wg := &sync.WaitGroup{}
	wg.Add(1)
	quit := make(chan *sync.WaitGroup, 1)
	go r.SubscribeEvent("foo", func() {
		fmt.Println("start")
	},
		func(channel string, data string) {
			handled = true
			Expect(channel).To(Equal("foo"))
			Expect(data).To(Equal("set"))
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	err = r.Set("foo", "bar")
	Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 200)
	Expect(handled).To(BeTrue())
	quit <- wg
	wg.Wait()

	err = r.Del("faz")
	Expect(err).To(BeNil())
	handled = false
	wg.Add(1)
	go r.SubscribeEvent("faz", func() {
		fmt.Println("start")
	},
		func(channel string, event string) {
			handled = true
			Expect(channel).To(Equal("faz"))
			Expect(event).To(Equal("set"))
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	err = r.Set("faz", "bar")
	Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 200)
	Expect(handled).To(BeTrue())
	quit <- wg
	wg.Wait()

	err = r.Del("bar")
	Expect(err).To(BeNil())
	handled = false
	wg.Add(1)
	go r.SubscribeEvent("bar", func() {
		fmt.Println("start")
	},
		func(channel string, event string) {
			handled = true
			Expect(channel).To(Equal("bar"))
			Expect(event).To(Equal("hset"))
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	err = r.HSet("bar", "baz", "baa")
	Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 200)
	Expect(handled).To(BeTrue())
	quit <- wg
	wg.Wait()
}

func TestExpirePersist(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "127.0.0.1:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("foo")
	Expect(err).To(BeNil())
	err = r.Set("foo", "bar")
	Expect(err).To(BeNil())
	v, err := r.Get("foo")
	Expect(err).To(BeNil())
	Expect(v).To(Equal("bar"))
	err = r.Expire("foo", time.Millisecond*10)
	Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 11)
	v, err = r.Get("foo")
	Expect(err).To(Equal(redis.ErrNil))
	Expect(v).To(BeEmpty())
	err = r.Set("foo", "bar")
	Expect(err).To(BeNil())
	err = r.Expire("foo", time.Millisecond*10)
	Expect(err).To(BeNil())
	time.Sleep(time.Millisecond)
	v, err = r.Get("foo")
	Expect(err).To(BeNil())
	Expect(v).To(Equal("bar"))
	err = r.Persist("foo")
	Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 20)
	v, err = r.Get("foo")
	Expect(err).To(BeNil())
	Expect(v).To(Equal("bar"))
}

func TestSets(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "127.0.0.1:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("*")
	Expect(err).To(BeNil())

	err = r.SAdd("set1", "value1")
	Expect(err).To(BeNil())
	v, err := r.SIsMember("set1", "value1")
	Expect(err).To(BeNil())
	Expect(v).To(BeTrue())
	err = r.SRem("set1", "value1")
	Expect(err).To(BeNil())
	v, err = r.SIsMember("set1", "value1")
	Expect(err).To(BeNil())
	Expect(v).To(BeFalse())

	err = r.SAdd("set1", "value1")
	Expect(err).To(BeNil())
	err = r.SAdd("set1", "value2")
	Expect(err).To(BeNil())
	err = r.SAdd("set1", "value3")
	Expect(err).To(BeNil())
	err = r.SAdd("set1", "value4")
	Expect(err).To(BeNil())
	members, err := r.SMembers("set1")
	Expect(err).To(BeNil())
	log.Println(members)
	Expect(len(members)).To(Equal(4))
}

func TestUnixSocket(t *testing.T) {
	t.Skip("manual")
	RegisterTestingT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "/var/run/redis/redis-server.sock",
		Net:     "unix",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Set("unix_test", "hello")
	Expect(err).To(BeNil())
	v, err := r.Get("unix_test")
	Expect(err).To(BeNil())
	Expect(v).To(Equal("hello"))
}

func TestStream(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "127.0.0.1:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("*")
	Expect(err).To(BeNil())

	streamName := "stream1"
	items := []StreamItem{
		{"", "key1", "value1"},
		{"", "key2", "value2"},
		{"", "key3", "value3"},
		{"", "key4", "value4"},
	}
	go func() {
		for i := range items {
			_, err := r.XAdd(streamName, items[i])
			Expect(err).To(BeNil())
		}
	}()

	var (
		res    []StreamItem
		lastID string = "0"
	)
	for i := 0; i < len(items); {
		kv, err := r.XRead(streamName, lastID)
		Expect(err).To(BeNil())
		res = append(res, kv...)
		i += len(kv)
		lastID = kv[len(kv)-1].ID
	}
	Expect(len(res)).To(Equal(len(items)))
	for i, item := range items {
		Expect(res[i].Key).To(Equal(item.Key))
		Expect(res[i].Value).To(Equal(item.Value))
	}
}

func TestTransaction(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "127.0.0.1:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("*")
	Expect(err).To(BeNil())

	err = r.Start().
		Set("x", "y").
		Set("x", "z").
		HSet("h", "x", "1").
		HSet("h", "x", "2").
		SAdd("s", "a").
		SAdd("s", "b").
		SRem("s", "a").
		Set("z", "1").
		Del("z").
		Commit()

	Expect(err).To(BeNil())
	x, err := r.Get("x")
	Expect(err).To(BeNil())
	Expect(x).To(Equal("z"))
	hx, err := r.HGet("h", "x")
	Expect(err).To(BeNil())
	Expect(hx).To(Equal("2"))
	sa, err := r.SIsMember("s", "a")
	Expect(err).To(BeNil())
	Expect(sa).To(BeFalse())
	sb, err := r.SIsMember("s", "b")
	Expect(err).To(BeNil())
	Expect(sb).To(BeTrue())
	z, err := r.Get("z")
	Expect(z).To(BeEmpty())
}

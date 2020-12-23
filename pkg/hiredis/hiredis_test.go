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
	g := NewGomegaWithT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("*")
	g.Expect(err).To(BeNil())

	err = r.Del("1")
	g.Expect(err).To(BeNil())
	err = r.Set("1", "1")
	g.Expect(err).To(BeNil())
	v, err := r.Get("1")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(Equal("1"))

	err = r.Del("2")
	g.Expect(err).To(BeNil())
	err = r.HSet("2", "key1", "value1")
	g.Expect(err).To(BeNil())
	err = r.HSet("2", "key2", "value2")
	g.Expect(err).To(BeNil())
	hkeys, err := r.GetHKeys("2")
	g.Expect(err).To(BeNil())
	g.Expect((hkeys[0] == "key1" && hkeys[1] == "key2") || (hkeys[0] == "key2" && hkeys[1] == "key1")).To(Equal(true))

	v, err = r.HGet("2", "key1")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(Equal("value1"))
	v, err = r.HGet("2", "key2")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(Equal("value2"))

	l, err := r.GetKeys("*")
	g.Expect(err).To(BeNil())
	g.Expect(len(l)).To(Equal(2))
	err = r.Del("*")
	g.Expect(err).To(BeNil())
	l, err = r.GetKeys("*")
	g.Expect(err).To(BeNil())
	g.Expect(len(l)).To(Equal(0))
}

func TestConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	cfg := Config{
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)

	err := r.SetConfig("notify-keyspace-events", "AE")
	g.Expect(err).To(BeNil())
	config, err := r.GetConfig("notify-keyspace-events")
	g.Expect(err).To(BeNil())
	g.Expect(config).To(Equal("AE"))
	err = r.SetConfig("notify-keyspace-events", "AKE")
	g.Expect(err).To(BeNil())
	config, err = r.GetConfig("notify-keyspace-events")
	g.Expect(err).To(BeNil())
	g.Expect(config).To(Equal("AKE"))
}

func TestPubSub(t *testing.T) {
	g := NewGomegaWithT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)

	err := r.Del("foo")
	g.Expect(err).To(BeNil())
	handled := false
	wg := &sync.WaitGroup{}
	wg.Add(1)
	quit := make(chan *sync.WaitGroup, 1)
	go r.SubscribeEvent("foo", func() {
		fmt.Println("start")
	},
		func(channel string, data string) {
			handled = true
			g.Expect(channel).To(Equal("foo"))
			g.Expect(data).To(Equal("set"))
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	err = r.Set("foo", "bar")
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 200)
	g.Expect(handled).To(BeTrue())
	quit <- wg
	wg.Wait()

	err = r.Del("faz")
	g.Expect(err).To(BeNil())
	handled = false
	wg.Add(1)
	go r.SubscribeEvent("faz", func() {
		fmt.Println("start")
	},
		func(channel string, event string) {
			handled = true
			g.Expect(channel).To(Equal("faz"))
			g.Expect(event).To(Equal("set"))
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	err = r.Set("faz", "bar")
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 200)
	g.Expect(handled).To(BeTrue())
	quit <- wg
	wg.Wait()

	err = r.Del("bar")
	g.Expect(err).To(BeNil())
	handled = false
	wg.Add(1)
	go r.SubscribeEvent("bar", func() {
		fmt.Println("start")
	},
		func(channel string, event string) {
			handled = true
			g.Expect(channel).To(Equal("bar"))
			g.Expect(event).To(Equal("hset"))
		}, func(err error) {
			fmt.Println(err)
		}, quit)
	time.Sleep(time.Millisecond * 200)
	err = r.HSet("bar", "baz", "baa")
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 200)
	g.Expect(handled).To(BeTrue())
	quit <- wg
	wg.Wait()
}

func TestExpirePersist(t *testing.T) {
	g := NewGomegaWithT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("foo")
	g.Expect(err).To(BeNil())
	err = r.Set("foo", "bar")
	g.Expect(err).To(BeNil())
	v, err := r.Get("foo")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(Equal("bar"))
	err = r.Expire("foo", time.Millisecond*10)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 11)
	v, err = r.Get("foo")
	g.Expect(err).To(Equal(redis.ErrNil))
	g.Expect(v).To(BeEmpty())
	err = r.Set("foo", "bar")
	g.Expect(err).To(BeNil())
	err = r.Expire("foo", time.Millisecond*10)
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond)
	v, err = r.Get("foo")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(Equal("bar"))
	err = r.Persist("foo")
	g.Expect(err).To(BeNil())
	time.Sleep(time.Millisecond * 11)
	v, err = r.Get("foo")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(Equal("bar"))
}

func TestSets(t *testing.T) {
	g := NewGomegaWithT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "redis:6379",
		Net:     "tcp",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Del("*")
	g.Expect(err).To(BeNil())

	err = r.SAdd("set1", "value1")
	g.Expect(err).To(BeNil())
	v, err := r.SIsMember("set1", "value1")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(BeTrue())
	err = r.SRem("set1", "value1")
	g.Expect(err).To(BeNil())
	v, err = r.SIsMember("set1", "value1")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(BeFalse())

	err = r.SAdd("set1", "value1")
	g.Expect(err).To(BeNil())
	err = r.SAdd("set1", "value2")
	g.Expect(err).To(BeNil())
	err = r.SAdd("set1", "value3")
	g.Expect(err).To(BeNil())
	err = r.SAdd("set1", "value4")
	g.Expect(err).To(BeNil())
	members, err := r.SMembers("set1")
	g.Expect(err).To(BeNil())
	log.Println(members)
	g.Expect(len(members)).To(Equal(4))
}

func TestUnixSocket(t *testing.T) {
	g := NewGomegaWithT(t)
	cfg := Config{
		Suffix:  "_redistest",
		Prefix:  "redistest_",
		Address: "/var/run/redis/redis-server.sock",
		Net:     "unix",
		DB:      0,
	}
	r := NewRedis(&cfg)
	err := r.Set("unix_test", "hello")
	g.Expect(err).To(BeNil())
	v, err := r.Get("unix_test")
	g.Expect(err).To(BeNil())
	g.Expect(v).To(Equal("hello"))
}

package ratelimit

import (
	"fmt"
	. "github.com/onsi/gomega"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	RegisterTestingT(t)
	cfg := Config{
		Enable:    true,
		Rate:      60000,
		Burst:     10,
		WhiteList: []string{"w1", "w2"},
		BlackList: []string{"b1", "b2"},
	}
	rl := NewRateLimiter(&cfg)

	fail := 0
	success := 0
	for i := 0; i < 10; i++ {
		if rl.CanHandle("1") == false {
			fail++
		} else {
			success++
		}
	}
	fmt.Println("fail : ", fail, " success : ", success)

	Expect(fail).To(Equal(0))
	fail = 0
	success = 0
	for i := 0; i < 20; i++ {
		if rl.CanHandle("2") == false {
			fail++
		} else {
			success++
		}
	}
	fmt.Println("fail : ", fail, " success : ", success)
	Expect(fail).To(Equal(9))
	Expect(success).To(Equal(11))

	Expect(rl.CanHandle("b1")).To(BeFalse())
	Expect(rl.CanHandle("b2")).To(BeFalse())

	for i := 0; i < 100; i++ {
		Expect(rl.CanHandle("w1")).To(BeTrue())
		Expect(rl.CanHandle("w2")).To(BeTrue())
	}

	fail = 0
	success = 0
	for i := 0; i < 10; i++ {
		Expect(rl.CanHandle("3")).To(BeTrue())
	}
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond)
		Expect(rl.CanHandle("3")).To(BeTrue())
	}
	fmt.Println("fail : ", fail, " success : ", success)
}

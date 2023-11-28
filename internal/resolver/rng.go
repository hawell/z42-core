package resolver

import "time"

func LehmerRandom(max int) int {
	return int(uint64(time.Now().Nanosecond())*48271%0x7fffffff) % max
}

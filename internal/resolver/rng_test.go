package resolver

import (
	. "github.com/onsi/gomega"
	"math"
	"testing"
)

func standardDeviation(num []int) float64 {
	var sum, mean, sd float64
	for i := 0; i < len(num); i++ {
		sum += float64(num[i])
	}
	mean = sum / float64(len(num))
	for j := 0; j < len(num); j++ {
		sd += math.Pow(float64(num[j])-mean, 2)
	}
	sd = math.Sqrt(sd / float64(len(num)))
	return sd
}

func TestLehmerRandom(t *testing.T) {
	RegisterTestingT(t)
	max := 1000
	dist := make([]int, max)
	for i := 0; i < 1000000; i++ {
		dist[LehmerRandom(max)]++
	}
	Expect(standardDeviation(dist)).To(BeNumerically("<", 100))
}

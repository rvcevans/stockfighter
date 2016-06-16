package quantest

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

func TestCKMS(t *testing.T) {
	const windowSize = 1000000

	quantiles := []Quantile{
		NewQuantile(0.5, 0.05),
		NewQuantile(0.9, 0.01),
		NewQuantile(0.95, 0.005),
		NewQuantile(0.99, 0.001),
	}

	estimator := NewCKMS(quantiles)
	samples := make([]int, windowSize, windowSize)

	rand.Seed(0xDEADBEEF)
	for i := 0; i < windowSize; i++ {
		r := int(rand.Int31n(100000))
		samples[i] = r
		estimator.Insert(r)
	}

	sort.Ints(samples)

	for _, quantile := range quantiles {
		estimate := estimator.Query(quantile.q)
		actual := samples[int(windowSize*quantile.q)-1]
		err := math.Abs(float64(estimate-actual)) / float64(actual)
		t.Logf("%s: estimate=%d, actual=%d, off=%.3f. memory saving=%.2f", quantile, estimate, actual, err, float64(estimator.len())/windowSize)
		if err > quantile.errFrac {
			t.Errorf("error in quantile estimation too large: max %.3f, got %.3f", quantile.errFrac, err)
		}
	}
}

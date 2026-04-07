package dicestats

import (
	"math"
	"sort"
)

const probabilityEpsilon = 1e-15

type Distribution struct {
	pmf         map[int]float64
	keys        []int // sorted outcome keys, computed once at construction
	approximate bool
}

func newDistribution(pmf map[int]float64, approximate bool) *Distribution {
	norm := normalizePMF(pmf)
	return newDistributionFromNormalized(norm, approximate)
}

// newDistributionExact builds a Distribution from exact arithmetic.
// Skips the epsilon filter and map copy, but still
// renormalizes to correct floating-point drift from convolution/multiplication.
func newDistributionExact(pmf map[int]float64) *Distribution {
	sum := 0.0
	c := 0.0
	for _, v := range pmf {
		y := v - c
		t := sum + y
		c = (t - sum) - y
		sum = t
	}
	if sum > 0 && sum != 1.0 {
		for k, v := range pmf {
			pmf[k] = v / sum
		}
	}
	return newDistributionFromNormalized(pmf, false)
}

func newDistributionFromNormalized(pmf map[int]float64, approximate bool) *Distribution {
	keys := make([]int, 0, len(pmf))
	for k := range pmf {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return &Distribution{pmf: pmf, keys: keys, approximate: approximate}
}

func normalizePMF(pmf map[int]float64) map[int]float64 {
	out := make(map[int]float64, len(pmf))
	sum := kahanSumValues(pmf)
	for k, v := range pmf {
		if v <= probabilityEpsilon {
			continue
		}
		out[k] = v
	}
	if sum <= probabilityEpsilon {
		return map[int]float64{0: 1}
	}
	for k, v := range out {
		out[k] = v / sum
	}
	return out
}

func (d *Distribution) PMF() map[int]float64 {
	cp := make(map[int]float64, len(d.pmf))
	for k, v := range d.pmf {
		cp[k] = v
	}
	return cp
}

func (d *Distribution) Expected() float64 {
	e := 0.0
	c := 0.0
	for x, p := range d.pmf {
		y := float64(x)*p - c
		t := e + y
		c = (t - e) - y
		e = t
	}
	return e
}

func (d *Distribution) Variance() float64 {
	mu := d.Expected()
	v := 0.0
	c := 0.0
	for x, p := range d.pmf {
		delta := float64(x) - mu
		y := delta*delta*p - c
		t := v + y
		c = (t - v) - y
		v = t
	}
	return v
}

func (d *Distribution) StdDev() float64 {
	return math.Sqrt(d.Variance())
}

func (d *Distribution) Min() int {
	if len(d.keys) == 0 {
		return 0
	}
	return d.keys[0]
}

func (d *Distribution) Max() int {
	if len(d.keys) == 0 {
		return 0
	}
	return d.keys[len(d.keys)-1]
}

func (d *Distribution) Prob(cmp Cmp, value float64) float64 {
	total := 0.0
	c := 0.0
	for x, p := range d.pmf {
		if compare(float64(x), cmp, value) {
			y := p - c
			t := total + y
			c = (t - total) - y
			total = t
		}
	}
	return total
}

func (d *Distribution) Percentile(p float64) int {
	if p <= 0 {
		return d.Min()
	}
	if p >= 1 {
		return d.Max()
	}
	cum := 0.0
	for _, k := range d.keys {
		cum += d.pmf[k]
		if cum >= p {
			return k
		}
	}
	return d.Max()
}

func (d *Distribution) Mode() int {
	bestK := 0
	bestP := -1.0
	// Return the smallest outcome when there are multiple modes.
	for k, p := range d.pmf {
		if p > bestP || (p == bestP && k < bestK) {
			bestK, bestP = k, p
		}
	}
	return bestK
}

func (d *Distribution) Median() int {
	return d.Percentile(0.5)
}

func (d *Distribution) Approximate() bool {
	return d.approximate
}

func compare(left float64, cmp Cmp, right float64) bool {
	switch cmp {
	case CmpGT:
		return left > right
	case CmpGTE:
		return left >= right
	case CmpLT:
		return left < right
	case CmpLTE:
		return left <= right
	case CmpEQ:
		return left == right
	case CmpNE:
		return left != right
	default:
		return false
	}
}

func kahanSumValues(pmf map[int]float64) float64 {
	sum := 0.0
	c := 0.0
	for _, v := range pmf {
		if v <= probabilityEpsilon {
			continue
		}
		y := v - c
		t := sum + y
		c = (t - sum) - y
		sum = t
	}
	return sum
}

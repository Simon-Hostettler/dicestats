package dicestats

import (
	"fmt"
	"math"
)

func normalizedKeepCount(kind keepDropKind, n, count int) (int, error) {
	keep := n
	switch kind {
	case dropHighest, dropLowest:
		keep = count - n
	}
	if keep < 0 || keep > count {
		return 0, fmt.Errorf("invalid keep size")
	}
	return keep, nil
}

func keptSumFromFaceCounts(counts []int, kind keepDropKind, keep int) int {
	if keep <= 0 {
		return 0
	}
	sum := 0
	remaining := keep
	if kind == keepHighest || kind == dropLowest {
		for face := len(counts) - 1; face >= 1 && remaining > 0; face-- {
			take := min(remaining, counts[face])
			sum += face * take
			remaining -= take
		}
		return sum
	}
	for face := 1; face < len(counts) && remaining > 0; face++ {
		take := min(remaining, counts[face])
		sum += face * take
		remaining -= take
	}
	return sum
}

func multinomialCount(total int, buckets []int) float64 {
	remaining := total
	ways := 1.0
	for _, c := range buckets {
		if c < 0 || c > remaining {
			return 0
		}
		ways *= binomialFloat(remaining, c)
		remaining -= c
	}
	if remaining != 0 {
		return 0
	}
	return ways
}

func binomialFloat(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n-k {
		k = n - k
	}
	res := 1.0
	for i := 1; i <= k; i++ {
		res *= float64(n-k+i) / float64(i)
	}
	return res
}

func applyBinaryOp(op binaryOp, left, right int) (int, error) {
	switch op {
	case opAdd:
		return left + right, nil
	case opSub:
		return left - right, nil
	case opMul:
		return left * right, nil
	default:
		return 0, fmt.Errorf("unsupported binary op")
	}
}

func convolveDistributionTimes(base *Distribution, n int) *Distribution {
	if n <= 0 {
		return newDistribution(map[int]float64{0: 1}, base.approximate)
	}
	pmf := map[int]float64{0: 1}
	for i := 0; i < n; i++ {
		next := make(map[int]float64)
		for lv, lp := range pmf {
			for rv, rp := range base.pmf {
				next[lv+rv] += lp * rp
			}
		}
		pmf = next
	}
	return newDistribution(pmf, base.approximate)
}

func saturatingBinomial(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n-k {
		k = n - k
	}
	lhs := 0.0
	for i := 1; i <= k; i++ {
		lhs += math.Log(float64(n-k+i)) - math.Log(float64(i))
	}
	if lhs >= math.Log(float64(math.MaxInt)) {
		return math.MaxInt
	}
	return int(math.Round(math.Exp(lhs)))
}

package dicestats

import (
	"fmt"
	"math/rand"
	"time"
)

func simulateExpr(expr expr, cfg *config) (*Distribution, error) {
	if cfg.simulationSamples <= 0 {
		return nil, fmt.Errorf("simulation samples must be > 0")
	}
	seed := time.Now().UnixNano()
	if cfg.simulationSeed != nil {
		seed = *cfg.simulationSeed
	}
	rng := rand.New(rand.NewSource(seed))
	counts := map[int]int{}
	for i := 0; i < cfg.simulationSamples; i++ {
		v, err := sampleExpr(expr, rng, cfg)
		if err != nil {
			return nil, err
		}
		counts[v]++
	}
	pmf := make(map[int]float64, len(counts))
	for k, c := range counts {
		pmf[k] = float64(c) / float64(cfg.simulationSamples)
	}
	return newDistribution(pmf, true), nil
}

func keptSumFromSorted(rolls []int, kind keepDropKind, keep int) int {
	n := len(rolls)
	if keep <= 0 {
		return 0
	}
	sum := 0
	switch kind {
	case keepHighest, dropLowest:
		for i := n - keep; i < n; i++ {
			if i >= 0 {
				sum += rolls[i]
			}
		}
	case keepLowest, dropHighest:
		for i := 0; i < keep && i < n; i++ {
			sum += rolls[i]
		}
	}
	return sum
}

func sampleExpr(expr expr, rng *rand.Rand, cfg *config) (int, error) {
	return expr.sample(rng, cfg)
}

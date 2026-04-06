package dicestats

import (
	"fmt"
	"math/rand"
	"sort"
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

func sortedCopy(in []int) []int {
	out := append([]int(nil), in...)
	sort.Ints(out)
	return out
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
	switch e := expr.(type) {
	case *numberExpr:
		return e.Value, nil
	case *diceExpr:
		if e.Count <= 0 || e.Sides <= 0 {
			return 0, fmt.Errorf("invalid dice")
		}
		sum := 0
		for i := 0; i < e.Count; i++ {
			sum += rng.Intn(e.Sides) + 1
		}
		return sum, nil
	case *repeatExpr:
		if e.Count <= 0 {
			return 0, fmt.Errorf("invalid repeat count")
		}
		sum := 0
		for i := 0; i < e.Count; i++ {
			v, err := sampleExpr(e.Base, rng, cfg)
			if err != nil {
				return 0, err
			}
			sum += v
		}
		return sum, nil
	case *binaryExpr:
		l, err := sampleExpr(e.Left, rng, cfg)
		if err != nil {
			return 0, err
		}
		r, err := sampleExpr(e.Right, rng, cfg)
		if err != nil {
			return 0, err
		}
		return applyBinaryOp(e.Op, l, r)
	case *keepDropExpr:
		d, ok := e.Base.(*diceExpr)
		if !ok {
			return 0, fmt.Errorf("keep/drop only supported on dice")
		}
		rolls := make([]int, d.Count)
		for i := 0; i < d.Count; i++ {
			rolls[i] = rng.Intn(d.Sides) + 1
		}
		keep, err := normalizedKeepCount(e.Kind, e.N, d.Count)
		if err != nil {
			return 0, err
		}
		return keptSumFromSorted(sortedCopy(rolls), e.Kind, keep), nil
	case *funcExpr:
		switch e.Kind {
		case functionMax:
			a, err := sampleExpr(e.First, rng, cfg)
			if err != nil {
				return 0, err
			}
			b, err := sampleExpr(e.Second, rng, cfg)
			if err != nil {
				return 0, err
			}
			if a > b {
				return a, nil
			}
			return b, nil
		case functionMin:
			a, err := sampleExpr(e.First, rng, cfg)
			if err != nil {
				return 0, err
			}
			b, err := sampleExpr(e.Second, rng, cfg)
			if err != nil {
				return 0, err
			}
			if a < b {
				return a, nil
			}
			return b, nil
		case functionBest, functionAdv, functionWorst, functionDis:
			best, err := sampleExpr(e.First, rng, cfg)
			if err != nil {
				return 0, err
			}
			for i := 1; i < e.N; i++ {
				v, err := sampleExpr(e.First, rng, cfg)
				if err != nil {
					return 0, err
				}
				if e.Kind == functionBest || e.Kind == functionAdv {
					if v > best {
						best = v
					}
				} else {
					if v < best {
						best = v
					}
				}
			}
			return best, nil
		default:
			return 0, fmt.Errorf("unknown function %s", e.Name)
		}
	case *probExpr:
		d, err := evalMaybeSim(e.Inner, cfg)
		if err != nil {
			return 0, err
		}
		p := d.Prob(e.Cmp, e.Value)
		if rng.Float64() < p {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported expression type %T", expr)
	}
}

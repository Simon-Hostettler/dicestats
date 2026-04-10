package dicestats

import (
	"fmt"
	"math/rand"
	"sort"
)

// --- numberExpr ---

func (e *numberExpr) eval(cfg *config) (*Distribution, error) {
	return newDistribution(map[int]float64{e.Value: 1}, false), nil
}

func (e *numberExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
	return e.Value, nil
}

func (e *numberExpr) estimateCost() int {
	return 1
}

// --- diceExpr ---

func (e *diceExpr) eval(cfg *config) (*Distribution, error) {
	if e.Count <= 0 || e.Sides <= 0 {
		return nil, fmt.Errorf("invalid dice %dd%d", e.Count, e.Sides)
	}
	return evalDice(e.Count, e.Sides), nil
}

func (e *diceExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
	if e.Count <= 0 || e.Sides <= 0 {
		return 0, fmt.Errorf("invalid dice")
	}
	sum := 0
	for i := 0; i < e.Count; i++ {
		sum += rng.Intn(e.Sides) + 1
	}
	return sum, nil
}

func (e *diceExpr) estimateCost() int {
	if e.Count <= 0 || e.Sides <= 0 {
		return 0
	}
	return estimateDiceConvolutionCost(e.Count, e.Sides)
}

// --- repeatExpr ---

func (e *repeatExpr) eval(cfg *config) (*Distribution, error) {
	if e.Count <= 0 {
		return nil, fmt.Errorf("invalid repeat count %d", e.Count)
	}
	base, err := evalExpr(e.Base, cfg)
	if err != nil {
		return nil, err
	}
	return convolveDistributionTimes(base, e.Count), nil
}

func (e *repeatExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
	if e.Count <= 0 {
		return 0, fmt.Errorf("invalid repeat count")
	}
	sum := 0
	for i := 0; i < e.Count; i++ {
		v, err := e.Base.sample(rng, cfg)
		if err != nil {
			return 0, err
		}
		sum += v
	}
	return sum, nil
}

func (e *repeatExpr) estimateCost() int {
	if e.Count <= 0 {
		return 0
	}
	base := e.Base.estimateCost()
	if base <= 0 {
		return 0
	}
	acc := base
	for i := 1; i < e.Count; i++ {
		acc = saturatingAdd(acc, base-1)
	}
	return acc
}

// --- binaryExpr ---

func (e *binaryExpr) eval(cfg *config) (*Distribution, error) {
	return evalBinary(e, cfg)
}

func (e *binaryExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
	l, err := e.Left.sample(rng, cfg)
	if err != nil {
		return 0, err
	}
	r, err := e.Right.sample(rng, cfg)
	if err != nil {
		return 0, err
	}
	return applyBinaryOp(e.Op, l, r)
}

func (e *binaryExpr) estimateCost() int {
	l := e.Left.estimateCost()
	r := e.Right.estimateCost()
	switch e.Op {
	case opAdd, opSub:
		if l <= 0 || r <= 0 {
			return 0
		}
		return saturatingAdd(l, r-1)
	case opMul:
		return saturatingMul(l, r)
	default:
		return saturatingMul(l, r)
	}
}

// --- keepDropExpr ---

func (e *keepDropExpr) eval(cfg *config) (*Distribution, error) {
	return evalKeepDrop(e, cfg)
}

func (e *keepDropExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
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
	sorted := append([]int(nil), rolls...)
	sort.Ints(sorted)
	return keptSumFromSorted(sorted, e.Kind, keep), nil
}

func (e *keepDropExpr) estimateCost() int {
	if d, ok := e.Base.(*diceExpr); ok && d.Count > 0 && d.Sides > 0 {
		if _, err := normalizedKeepCount(e.Kind, e.N, d.Count); err != nil {
			return 0
		}
		return saturatingBinomial(d.Count+d.Sides-1, d.Sides-1)
	}
	return e.Base.estimateCost()
}

// --- binaryFuncExpr ---

func (e *binaryFuncExpr) eval(cfg *config) (*Distribution, error) {
	return evalBinaryFunc(e, cfg)
}

func (e *binaryFuncExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
	a, err := e.Left.sample(rng, cfg)
	if err != nil {
		return 0, err
	}
	b, err := e.Right.sample(rng, cfg)
	if err != nil {
		return 0, err
	}
	if e.Kind == functionMax {
		if a > b {
			return a, nil
		}
		return b, nil
	}
	if a < b {
		return a, nil
	}
	return b, nil
}

func (e *binaryFuncExpr) estimateCost() int {
	a := e.Left.estimateCost()
	b := e.Right.estimateCost()
	if a <= 0 || b <= 0 {
		return 0
	}
	return max(a, b)
}

// --- orderStatExpr ---

func (e *orderStatExpr) eval(cfg *config) (*Distribution, error) {
	return evalOrderStat(e, cfg)
}

func (e *orderStatExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
	best, err := e.Base.sample(rng, cfg)
	if err != nil {
		return 0, err
	}
	for i := 1; i < e.N; i++ {
		v, err := e.Base.sample(rng, cfg)
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
}

func (e *orderStatExpr) estimateCost() int {
	base := e.Base.estimateCost()
	if e.Kind == functionAdv || e.Kind == functionDis {
		return base
	}
	return saturatingAdd(base, e.N-1)
}

// --- indicatorExpr ---

func (e *indicatorExpr) eval(cfg *config) (*Distribution, error) {
	return evalIndicator(e, cfg)
}

func (e *indicatorExpr) sample(rng *rand.Rand, cfg *config) (int, error) {
	d, err := evalMaybeSim(e.Inner, cfg)
	if err != nil {
		return 0, err
	}
	p := d.Prob(e.Cmp, e.Value)
	if rng.Float64() < p {
		return 1, nil
	}
	return 0, nil
}

func (e *indicatorExpr) estimateCost() int {
	base := e.Inner.estimateCost()
	if base <= 0 {
		return 0
	}
	return saturatingAdd(base, 1)
}

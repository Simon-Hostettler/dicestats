package dicestats

import (
	"fmt"
	"math"
	"strconv"
)

type config struct {
	simulationThreshold int
	simulationSamples   int
	simulationSeed      *int64
	cache               *Cache
}

type Option func(*config)

func defaultConfig() config {
	return config{
		simulationThreshold: 1_000_000,
		simulationSamples:   100_000,
		cache:               NewCache(),
	}
}

func WithSimulationThreshold(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.simulationThreshold = n
		}
	}
}

func WithSimulationSamples(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.simulationSamples = n
		}
	}
}

func WithSimulationSeed(seed int64) Option {
	return func(c *config) {
		c.simulationSeed = &seed
	}
}

func WithCache(cache *Cache) Option {
	return func(c *config) {
		if cache != nil {
			c.cache = cache
		}
	}
}

func eval(expr expr, opts ...Option) (*Distribution, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return evalMaybeSim(expr, &cfg)
}

func evalExpr(expr expr, cfg *config) (*Distribution, error) {
	key := cacheKey(expr.Key(), false, cfg)
	if d, ok := cfg.cache.Get(key); ok {
		return d, nil
	}
	out, err := evalExprUncached(expr, cfg)
	if err != nil {
		return nil, err
	}
	cfg.cache.Put(key, out)
	return out, nil
}

func evalExprUncached(expr expr, cfg *config) (*Distribution, error) {
	switch e := expr.(type) {
	case *numberExpr:
		return newDistribution(map[int]float64{e.Value: 1}, false), nil
	case *diceExpr:
		if e.Count <= 0 || e.Sides <= 0 {
			return nil, fmt.Errorf("invalid dice %dd%d", e.Count, e.Sides)
		}
		return evalDice(e.Count, e.Sides), nil
	case *repeatExpr:
		if e.Count <= 0 {
			return nil, fmt.Errorf("invalid repeat count %d", e.Count)
		}
		base, err := evalExpr(e.Base, cfg)
		if err != nil {
			return nil, err
		}
		return convolveDistributionTimes(base, e.Count), nil
	case *binaryExpr:
		return evalBinary(e, cfg)
	case *keepDropExpr:
		return evalKeepDrop(e)
	case *funcExpr:
		return evalFunc(e, cfg)
	case *probExpr:
		return evalProbGate(e, cfg)
	default:
		return nil, fmt.Errorf("unsupported expression type %T", expr)
	}
}

func evalMaybeSim(expr expr, cfg *config) (*Distribution, error) {
	simKey := cacheKey(expr.Key(), true, cfg)
	if d, ok := cfg.cache.Get(simKey); ok {
		return d, nil
	}
	exactKey := cacheKey(expr.Key(), false, cfg)
	if d, ok := cfg.cache.Get(exactKey); ok {
		return d, nil
	}
	if estimateEvaluationCost(expr) > cfg.simulationThreshold {
		d, err := simulateExpr(expr, cfg)
		if err != nil {
			return nil, err
		}
		cfg.cache.Put(simKey, d)
		return d, nil
	}
	// Cache already checked above; go straight to uncached eval + store.
	out, err := evalExprUncached(expr, cfg)
	if err != nil {
		return nil, err
	}
	cfg.cache.Put(exactKey, out)
	return out, nil
}

func evalDice(count, sides int) *Distribution {
	single := make(map[int]float64, sides)
	p := 1.0 / float64(sides)
	for face := 1; face <= sides; face++ {
		single[face] = p
	}
	return convolveDistributionTimes(newDistribution(single, false), count)
}

func evalBinary(e *binaryExpr, cfg *config) (*Distribution, error) {
	left, err := evalExpr(e.Left, cfg)
	if err != nil {
		return nil, err
	}
	right, err := evalExpr(e.Right, cfg)
	if err != nil {
		return nil, err
	}
	pmf := map[int]float64{}
	for lv, lp := range left.pmf {
		for rv, rp := range right.pmf {
			out, err := applyBinaryOp(e.Op, lv, rv)
			if err != nil {
				return nil, err
			}
			pmf[out] += lp * rp
		}
	}
	return newDistribution(pmf, left.approximate || right.approximate), nil
}

func evalKeepDrop(e *keepDropExpr) (*Distribution, error) {
	d, ok := e.Base.(*diceExpr)
	if !ok {
		return nil, fmt.Errorf("keep/drop modifiers only supported on dice expressions")
	}
	if d.Count <= 0 || d.Sides <= 0 {
		return nil, fmt.Errorf("invalid dice expression")
	}
	if e.N < 0 || e.N > d.Count {
		return nil, fmt.Errorf("invalid keep/drop count %d for %dd%d", e.N, d.Count, d.Sides)
	}

	keep, err := normalizedKeepCount(e.Kind, e.N, d.Count)
	if err != nil {
		return nil, err
	}

	pmf := map[int]float64{}
	total := math.Pow(float64(d.Sides), float64(d.Count))
	counts := make([]int, d.Sides+1)
	var rec func(face int, remaining int)
	rec = func(face int, remaining int) {
		if face == d.Sides {
			counts[face] = remaining
			ways := multinomialCount(d.Count, counts[1:])
			sum := keptSumFromFaceCounts(counts, e.Kind, keep)
			pmf[sum] += ways / total
			return
		}
		for c := 0; c <= remaining; c++ {
			counts[face] = c
			rec(face+1, remaining-c)
		}
	}
	rec(1, d.Count)

	return newDistribution(pmf, false), nil
}

func evalFunc(e *funcExpr, cfg *config) (*Distribution, error) {
	switch e.Kind {
	case functionMax, functionMin:
		a, err := evalExpr(e.First, cfg)
		if err != nil {
			return nil, err
		}
		b, err := evalExpr(e.Second, cfg)
		if err != nil {
			return nil, err
		}
		better := func(x, y int) bool { return x >= y }
		if e.Kind == functionMin {
			better = func(x, y int) bool { return x <= y }
		}
		pmf := map[int]float64{}
		for av, ap := range a.pmf {
			for bv, bp := range b.pmf {
				if better(av, bv) {
					pmf[av] += ap * bp
				} else {
					pmf[bv] += ap * bp
				}
			}
		}
		return newDistribution(pmf, a.approximate || b.approximate), nil
	case functionBest, functionAdv:
		base, err := evalExpr(e.First, cfg)
		if err != nil {
			return nil, err
		}
		return bestOf(e.N, base), nil
	case functionWorst, functionDis:
		base, err := evalExpr(e.First, cfg)
		if err != nil {
			return nil, err
		}
		return worstOf(e.N, base), nil
	default:
		return nil, fmt.Errorf("unsupported function %s", e.Name)
	}
}

func bestOf(n int, d *Distribution) *Distribution {
	keys := sortedKeys(d.pmf)
	cdf := 0.0
	prev := 0.0
	pmf := map[int]float64{}
	for _, k := range keys {
		cdf += d.pmf[k]
		curr := math.Pow(cdf, float64(n))
		pmf[k] = curr - prev
		prev = curr
	}
	return newDistribution(pmf, d.approximate)
}

func worstOf(n int, d *Distribution) *Distribution {
	keys := sortedKeys(d.pmf)
	pmf := map[int]float64{}
	cdf := 0.0
	prev := 0.0
	for _, k := range keys {
		cdf += d.pmf[k]
		curr := 1 - math.Pow(1-cdf, float64(n))
		pmf[k] = curr - prev
		prev = curr
	}
	return newDistribution(pmf, d.approximate)
}

func evalProbGate(e *probExpr, cfg *config) (*Distribution, error) {
	base, err := evalMaybeSim(e.Inner, cfg)
	if err != nil {
		return nil, err
	}
	p := base.Prob(e.Cmp, e.Value)
	return newDistribution(map[int]float64{0: 1 - p, 1: p}, base.approximate), nil
}

func estimateEvaluationCost(expr expr) int {
	switch e := expr.(type) {
	case *numberExpr:
		return 1
	case *diceExpr:
		if e.Count <= 0 || e.Sides <= 0 {
			return 0
		}
		// Work proxy for convolution DP in evalDice:
		// sum_{i=0}^{n-1} sides * (i*(sides-1)+1)
		// = sides * (n + (sides-1)*n*(n-1)/2)
		return estimateDiceConvolutionCost(e.Count, e.Sides)
	case *repeatExpr:
		if e.Count <= 0 {
			return 0
		}
		base := estimateEvaluationCost(e.Base)
		if base <= 0 {
			return 0
		}
		acc := base
		for i := 1; i < e.Count; i++ {
			acc = saturatingAdd(acc, base-1)
		}
		return acc
	case *binaryExpr:
		l := estimateEvaluationCost(e.Left)
		r := estimateEvaluationCost(e.Right)
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
	case *keepDropExpr:
		if d, ok := e.Base.(*diceExpr); ok && d.Count > 0 && d.Sides > 0 {
			if _, err := normalizedKeepCount(e.Kind, e.N, d.Count); err != nil {
				return 0
			}
			// Enumerates face-count compositions: C(count+sides-1, sides-1).
			return saturatingBinomial(d.Count+d.Sides-1, d.Sides-1)
		}
		return estimateEvaluationCost(e.Base)
	case *funcExpr:
		switch e.Kind {
		case functionAdv, functionDis:
			return estimateEvaluationCost(e.First)
		case functionBest, functionWorst:
			base := estimateEvaluationCost(e.First)
			return saturatingAdd(base, e.N-1)
		case functionMax, functionMin:
			a := estimateEvaluationCost(e.First)
			b := estimateEvaluationCost(e.Second)
			if a <= 0 || b <= 0 {
				return 0
			}
			return max(a, b)
		default:
			return 1
		}
	case *probExpr:
		base := estimateEvaluationCost(e.Inner)
		if base <= 0 {
			return 0
		}
		// Prob gate computes inner distribution once, then projects to Bernoulli.
		return saturatingAdd(base, 1)
	default:
		return 1
	}
}

func saturatingMul(a, b int) int {
	if a <= 0 || b <= 0 {
		return 0
	}
	if a > math.MaxInt/b {
		return math.MaxInt
	}
	return a * b
}

func saturatingAdd(a, b int) int {
	if a < 0 || b < 0 {
		return 0
	}
	if a > math.MaxInt-b {
		return math.MaxInt
	}
	return a + b
}

func estimateDiceConvolutionCost(count, sides int) int {
	if count <= 0 || sides <= 0 {
		return 0
	}
	iterations := saturatingMul(count, sides)
	if count == 1 {
		return iterations
	}
	countMinusOne := count - 1
	pairProduct := saturatingMul(count, countMinusOne)
	widthGrowth := saturatingMul(pairProduct, sides-1)
	widthGrowth /= 2
	inner := saturatingAdd(count, widthGrowth)
	return saturatingMul(sides, inner)
}

func cacheKey(exprKey string, modeAware bool, cfg *config) string {
	if !modeAware {
		return "exact:" + exprKey
	}
	return "sim:t=" + strconv.Itoa(cfg.simulationThreshold) + ":s=" + strconv.Itoa(cfg.simulationSamples) + ":" + exprKey
}

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
		c.simulationThreshold = n
	}
}

func WithSimulationSamples(n int) Option {
	return func(c *config) {
		c.simulationSamples = n
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
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return evalMaybeSim(expr, &cfg)
}

func (c *config) validate() error {
	if c.simulationThreshold <= 0 {
		return fmt.Errorf("simulation threshold must be > 0, got %d", c.simulationThreshold)
	}
	if c.simulationSamples <= 0 {
		return fmt.Errorf("simulation samples must be > 0, got %d", c.simulationSamples)
	}
	return nil
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
	return expr.eval(cfg)
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
	if left.approximate || right.approximate {
		return newDistribution(pmf, true), nil
	}
	return newDistributionExact(pmf), nil
}

func evalKeepDrop(e *keepDropExpr, cfg *config) (*Distribution, error) {
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

	return newDistributionExact(pmf), nil
}

func evalBinaryFunc(e *binaryFuncExpr, cfg *config) (*Distribution, error) {
	a, err := evalExpr(e.Left, cfg)
	if err != nil {
		return nil, err
	}
	b, err := evalExpr(e.Right, cfg)
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
	if a.approximate || b.approximate {
		return newDistribution(pmf, true), nil
	}
	return newDistributionExact(pmf), nil
}

func evalOrderStat(e *orderStatExpr, cfg *config) (*Distribution, error) {
	base, err := evalExpr(e.Base, cfg)
	if err != nil {
		return nil, err
	}
	if e.Kind == functionBest || e.Kind == functionAdv {
		return bestOf(e.N, base), nil
	}
	return worstOf(e.N, base), nil
}

func bestOf(n int, d *Distribution) *Distribution {
	cdf := 0.0
	prev := 0.0
	pmf := map[int]float64{}
	for _, k := range d.keys {
		cdf += d.pmf[k]
		curr := math.Pow(cdf, float64(n))
		pmf[k] = curr - prev
		prev = curr
	}
	if d.approximate {
		return newDistribution(pmf, true)
	}
	return newDistributionExact(pmf)
}

func worstOf(n int, d *Distribution) *Distribution {
	pmf := map[int]float64{}
	cdf := 0.0
	prev := 0.0
	for _, k := range d.keys {
		cdf += d.pmf[k]
		curr := 1 - math.Pow(1-cdf, float64(n))
		pmf[k] = curr - prev
		prev = curr
	}
	if d.approximate {
		return newDistribution(pmf, true)
	}
	return newDistributionExact(pmf)
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
	return expr.estimateCost()
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

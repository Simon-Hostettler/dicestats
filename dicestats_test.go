package dicestats

import (
	"math"
	"testing"
	"time"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func mustDist(t *testing.T, input string, opts ...Option) *Distribution {
	t.Helper()
	r, err := Query(input, opts...)
	if err != nil {
		t.Fatalf("Query(%q) error: %v", input, err)
	}
	if r.Distribution == nil {
		t.Fatalf("Query(%q) returned nil distribution", input)
	}
	return r.Distribution
}

func TestEval2d6Expected(t *testing.T) {
	d := mustDist(t, "2d6")
	if !almostEqual(d.Expected(), 7.0) {
		t.Fatalf("expected 7, got %f", d.Expected())
	}
}

func TestRepeatDrawsIndependentNotScaled(t *testing.T) {
	rep := mustDist(t, "3(max(3,1d6+1))")
	scaled := mustDist(t, "3*max(3,1d6+1)")
	if math.Abs(rep.Expected()-14.0) > 1e-12 {
		t.Fatalf("expected repeat EV 14.0, got %f", rep.Expected())
	}
	if math.Abs(scaled.Expected()-14.0) > 1e-12 {
		t.Fatalf("expected scaled EV 14.0, got %f", scaled.Expected())
	}
	if rep.Variance() >= scaled.Variance() {
		t.Fatalf("expected repeat variance < scaled variance, got %f >= %f", rep.Variance(), scaled.Variance())
	}
}

func TestRepeatDrawsOnDiceLiteralEquivalentToMultiDice(t *testing.T) {
	rep := mustDist(t, "3(1d6)")
	dice := mustDist(t, "3d6")
	repPMF := rep.PMF()
	dicePMF := dice.PMF()
	if len(repPMF) != len(dicePMF) {
		t.Fatalf("pmf size mismatch %d vs %d", len(repPMF), len(dicePMF))
	}
	for k, p := range dicePMF {
		if math.Abs(repPMF[k]-p) > 1e-12 {
			t.Fatalf("pmf mismatch at %d: got %f want %f", k, repPMF[k], p)
		}
	}
}

func TestDistributionMethodsOn1d6(t *testing.T) {
	d := mustDist(t, "1d6")
	if !almostEqual(d.Expected(), 3.5) {
		t.Fatalf("expected 3.5 got %f", d.Expected())
	}
	if !almostEqual(d.Variance(), 35.0/12.0) {
		t.Fatalf("expected variance 35/12 got %f", d.Variance())
	}
	if d.Min() != 1 || d.Max() != 6 {
		t.Fatalf("unexpected min/max %d..%d", d.Min(), d.Max())
	}
	if d.Mode() != 1 {
		t.Fatalf("expected mode 1 on flat distribution tie-break, got %d", d.Mode())
	}
	if d.Median() != 3 {
		t.Fatalf("expected median 3 got %d", d.Median())
	}
	if d.Percentile(0) != 1 || d.Percentile(1) != 6 {
		t.Fatalf("unexpected percentile bounds")
	}
	if p := d.Prob(CmpGTE, 4); math.Abs(p-0.5) > 1e-12 {
		t.Fatalf("expected P(>=4)=0.5 got %f", p)
	}
}

func TestKeepHighest(t *testing.T) {
	d := mustDist(t, "4d6kh3")
	if d.Min() != 3 || d.Max() != 18 {
		t.Fatalf("unexpected range %d..%d", d.Min(), d.Max())
	}
}

func TestKeepDropComplements(t *testing.T) {
	kh := mustDist(t, "4d6kh3")
	dl := mustDist(t, "4d6dl1")
	if math.Abs(kh.Expected()-dl.Expected()) > 1e-12 {
		t.Fatalf("kh3 and dl1 should match, got %f vs %f", kh.Expected(), dl.Expected())
	}

	kl := mustDist(t, "4d6kl3")
	dh := mustDist(t, "4d6dh1")
	if math.Abs(kl.Expected()-dh.Expected()) > 1e-12 {
		t.Fatalf("kl3 and dh1 should match, got %f vs %f", kl.Expected(), dh.Expected())
	}
}

func TestAdvantageBetterThanSingle(t *testing.T) {
	one := mustDist(t, "1d20")
	adv := mustDist(t, "adv(1d20)")
	if adv.Expected() <= one.Expected() {
		t.Fatalf("expected advantage EV > single EV")
	}
}

func TestWorstOfExpectedValue(t *testing.T) {
	dis := mustDist(t, "dis(1d20)")
	if math.Abs(dis.Expected()-7.175) > 1e-12 {
		t.Fatalf("expected 7.175 got %f", dis.Expected())
	}
}

func TestBestAndWorstOfThreeExpectedValues(t *testing.T) {
	best := mustDist(t, "best(3,1d20)")
	worst := mustDist(t, "worst(3,1d20)")
	if best.Expected() <= 13.825 {
		t.Fatalf("best(3,1d20) should beat advantage EV, got %f", best.Expected())
	}
	if worst.Expected() >= 7.175 {
		t.Fatalf("worst(3,1d20) should be below disadvantage EV, got %f", worst.Expected())
	}
}

func TestIndicatorDamage(t *testing.T) {
	r, err := Query("E[[1d20 + 10 > 15] * 5d6]")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(r.Value-13.125) > 1e-9 {
		t.Fatalf("expected 13.125 got %f", r.Value)
	}
}

func TestIndicatorDistributionShape(t *testing.T) {
	d := mustDist(t, "[1d20+10>15] * 1d4")
	pmf := d.PMF()
	if _, ok := pmf[0]; !ok {
		t.Fatal("expected miss branch at 0")
	}
	if pmf[0] <= 0 || pmf[0] >= 1 {
		t.Fatalf("unexpected miss probability %f", pmf[0])
	}
	if d.Min() != 0 || d.Max() != 4 {
		t.Fatalf("unexpected range %d..%d", d.Min(), d.Max())
	}
}

func TestProbQuery(t *testing.T) {
	r, err := Query("P[2d6 >= 10]")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(r.Value-(6.0/36.0)) > 1e-12 {
		t.Fatalf("unexpected probability %f", r.Value)
	}
}

func TestProbQueryRejectsMissingComparison(t *testing.T) {
	if _, err := Query("P[1d20]"); err == nil {
		t.Fatal("expected parse error for missing comparison in P query")
	}
}

func TestQueryWhitespaceTolerance(t *testing.T) {
	cases := []string{
		"E [ 2d6 + 3 ]",
		"Var [ 1d6 ]",
		"StdDev [ 1d6 ]",
		"D [ 2d6 ]",
		"P [ 2d6 >= 10 ]",
	}
	for _, tc := range cases {
		if _, err := Query(tc); err != nil {
			t.Fatalf("expected query %q to parse, got %v", tc, err)
		}
	}
}

func TestStatQueries(t *testing.T) {
	e, err := Query("E[2d6+3]")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(e.Value-10.0) > 1e-12 {
		t.Fatalf("expected 10 got %f", e.Value)
	}

	v, err := Query("Var[1d6]")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(v.Value-(35.0/12.0)) > 1e-12 {
		t.Fatalf("unexpected var %f", v.Value)
	}

	s, err := Query("StdDev[1d6]")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(s.Value-math.Sqrt(35.0/12.0)) > 1e-12 {
		t.Fatalf("unexpected stddev %f", s.Value)
	}
}

func TestDistAndBareExprQueries(t *testing.T) {
	r1, err := Query("D[2d6]")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Query("2d6")
	if err != nil {
		t.Fatal(err)
	}
	if r1.Distribution == nil || r2.Distribution == nil {
		t.Fatal("expected distributions")
	}
	if math.Abs(r1.Distribution.Expected()-r2.Distribution.Expected()) > 1e-12 {
		t.Fatalf("expected equal distributions")
	}
}

func TestClampFunctions(t *testing.T) {
	d := mustDist(t, "max(1, 1d4-2)")
	if d.Min() != 1 || d.Max() != 2 {
		t.Fatalf("unexpected max clamp range %d..%d", d.Min(), d.Max())
	}
	d2 := mustDist(t, "min(20, 1d20+5)")
	if d2.Min() != 6 || d2.Max() != 20 {
		t.Fatalf("unexpected min clamp range %d..%d", d2.Min(), d2.Max())
	}
}

func TestKeepDropGrammarIsDiceOnly(t *testing.T) {
	if _, err := parse("(4d6+0)kh3"); err == nil {
		t.Fatal("expected parser error for keep/drop on non-dice expression")
	}
	if _, err := parse("4d6kh3dl1"); err == nil {
		t.Fatal("expected parser error for multiple keep/drop modifiers")
	}
	if _, err := parse("adv(1d20)kh1"); err == nil {
		t.Fatal("expected parser error for keep/drop on function call")
	}
}

func TestKeepDropValidationErrors(t *testing.T) {
	if _, err := Query("4d6kh5"); err == nil {
		t.Fatal("expected invalid keep count error")
	}
	if _, err := Query("4d6dl5"); err == nil {
		t.Fatal("expected invalid drop count error")
	}
}

func TestEstimateAvoidsUnnecessarySimulation(t *testing.T) {
	d := mustDist(t, "best(5,1d20)", WithSimulationThreshold(25), WithSimulationSamples(2000))
	if d.Approximate() {
		t.Fatal("best-of should not force simulation based on sample-space explosion")
	}
}

func TestSimulationFallback(t *testing.T) {
	d := mustDist(t, "20d20*20d20", WithSimulationThreshold(1000), WithSimulationSamples(5000))
	if !d.Approximate() {
		t.Fatal("expected approximate distribution")
	}
}

func TestNestedIndicatorSimulationUsesConfig(t *testing.T) {
	d := mustDist(t, "[[20d20*20d20 > 20000] = 1] * 1d6",
		WithSimulationThreshold(1000),
		WithSimulationSamples(4000),
	)
	if !d.Approximate() {
		t.Fatal("expected approximate distribution with nested simulation")
	}
}

func TestKeepDropCompositionPathExact(t *testing.T) {
	d := mustDist(t, "8d6kh3", WithSimulationThreshold(5000), WithSimulationSamples(1000))
	if d.Approximate() {
		t.Fatal("expected exact distribution for composition keep/drop path")
	}
	if d.Min() != 3 || d.Max() != 18 {
		t.Fatalf("unexpected range %d..%d", d.Min(), d.Max())
	}
}

func TestDistributionNormalizationStable(t *testing.T) {
	d := newDistribution(map[int]float64{
		1: 0.1,
		2: 0.2,
		3: 0.7,
		4: 1e-16,
	}, false)
	pmf := d.PMF()
	total := 0.0
	for _, p := range pmf {
		total += p
	}
	if math.Abs(total-1) > 1e-12 {
		t.Fatalf("expected normalized total 1, got %f", total)
	}
	if _, ok := pmf[4]; ok {
		t.Fatal("expected tiny probability to be filtered")
	}
}

func TestNegativeOutcomeRange(t *testing.T) {
	d := mustDist(t, "1d6-10")
	if d.Min() != -9 || d.Max() != -4 {
		t.Fatalf("unexpected range %d..%d", d.Min(), d.Max())
	}
	if !almostEqual(d.Expected(), -6.5) {
		t.Fatalf("expected -6.5 got %f", d.Expected())
	}
}

func TestSimulationAccuracyFor2d6Expected(t *testing.T) {
	d := mustDist(t, "2d6", WithSimulationThreshold(1), WithSimulationSamples(120000), WithSimulationSeed(42))
	if !d.Approximate() {
		t.Fatal("expected approximate distribution")
	}
	if delta := math.Abs(d.Expected() - 7.0); delta > 0.08 {
		t.Fatalf("simulation expected value too far from exact: delta=%f", delta)
	}
}

func TestSharedCacheReuseAcrossQueries(t *testing.T) {
	cache := NewCache()
	if _, err := Query("E[4d6kh3]", WithCache(cache)); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.Get("exact:kh(d(4,6),3)"); !ok {
		t.Fatal("expected cached 4d6kh3 distribution")
	}
	if _, err := Query("P[4d6kh3 >= 16]", WithCache(cache)); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.Get("exact:kh(d(4,6),3)"); !ok {
		t.Fatal("expected cached 4d6kh3 distribution to remain available")
	}
}

func TestCacheSeparatesApproximateAndExactModes(t *testing.T) {
	cache := NewCache()
	approx := mustDist(t, "10d10*10d10", WithCache(cache), WithSimulationThreshold(1000), WithSimulationSamples(5000), WithSimulationSeed(7))
	if !approx.Approximate() {
		t.Fatal("expected approximate distribution in low-threshold mode")
	}
	exact := mustDist(t, "10d10*10d10", WithCache(cache), WithSimulationThreshold(20_000_000), WithSimulationSamples(5000), WithSimulationSeed(7))
	if exact.Approximate() {
		t.Fatal("expected exact distribution to not be shadowed by approximate cache")
	}
}

func TestParserRejectsWrongFunctionArity(t *testing.T) {
	cases := []string{"max(1)", "adv(1,2)", "best(1)"}
	for _, tc := range cases {
		if _, err := parse(tc); err == nil {
			t.Fatalf("expected parse error for %q", tc)
		}
	}
}

func TestSimulationStressLargeExpressionFastEnough(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	start := time.Now()
	d := mustDist(t, "40d40*40d40 + 30d30",
		WithSimulationThreshold(100),
		WithSimulationSamples(20000),
	)
	if !d.Approximate() {
		t.Fatal("expected simulation to be used for stress expression")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("simulation too slow: %s", elapsed)
	}
}

func TestSimulationStressBatchFastEnough(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	start := time.Now()
	for i := 0; i < 25; i++ {
		d := mustDist(t, "50d20*50d20",
			WithSimulationThreshold(100),
			WithSimulationSamples(2500),
		)
		if !d.Approximate() {
			t.Fatal("expected simulation in stress batch")
		}
	}
	if elapsed := time.Since(start); elapsed > 6*time.Second {
		t.Fatalf("batch simulation too slow: %s", elapsed)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		"d0",
		"0d6",
		"0(1d6)",
		"foo(1)",
		"2d6 +",
		"P[1d20 > ]",
	}
	for _, tc := range cases {
		if _, err := parse(tc); err == nil {
			t.Fatalf("expected parse error for %q", tc)
		}
	}
}

func TestQueryResultString(t *testing.T) {
	r, err := Query("E[2d6]")
	if err != nil {
		t.Fatal(err)
	}
	if s := r.String(); s != "7" {
		t.Fatalf("expected '7' got %q", s)
	}

	r2, err := Query("2d6")
	if err != nil {
		t.Fatal(err)
	}
	s2 := r2.String()
	if s2 == "" {
		t.Fatal("expected non-empty string for distribution result")
	}
}

func TestQueryTypeString(t *testing.T) {
	cases := map[QueryType]string{
		QueryExpected:    "E",
		QueryVariance:    "Var",
		QueryStdDev:      "StdDev",
		QueryDist:        "D",
		QueryProbability: "P",
	}
	for qt, want := range cases {
		if got := qt.String(); got != want {
			t.Fatalf("QueryType(%d).String() = %q, want %q", qt, got, want)
		}
	}
}

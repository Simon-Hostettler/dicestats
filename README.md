# dicestats

`dicestats` parses dice expressions and evaluates exact probability distributions (with simulation fallback for large spaces).

## Library API

Import path:

```go
import "hostettler.dev/dicestats"
```

- Use `Query` for query workflows (`E[...]`, `Var[...]`, `StdDev[...]`, `P[...]`, `D[...]`).
- Use `EvalString` when you want the full output distribution directly from an expression string.
- Use `WithCache(NewCache())` to reuse computation across calls.

Public entry points:

- `func EvalString(input string, opts ...dicestats.Option) (*dicestats.Distribution, error)`
- `func Query(input string, opts ...dicestats.Option) (*dicestats.QueryResult, error)`
- `type Distribution` with methods:
  - `PMF() map[int]float64`
  - `Expected() float64`
  - `Variance() float64`
  - `StdDev() float64`
  - `Min() int`, `Max() int`
  - `Median() int`, `Mode() int`, `Percentile(p float64) int`
  - `Prob(cmp dicestats.Cmp, value float64) float64`
  - `Approximate() bool`
- `type QueryResult` fields:
  - `Type dicestats.QueryType`
  - `Value float64`
  - `Distribution *dicestats.Distribution`
  - `Approximate bool`
- Options:
  - `WithSimulationThreshold(n int)`
  - `WithSimulationSamples(n int)`
  - `WithSimulationSeed(seed int64)`
  - `WithCache(cache *dicestats.Cache)`
- Caching:
  - `type Cache`
  - `func NewCache() *Cache`
- Comparators and query kinds:
  - `type Cmp` with `CmpGT`, `CmpGTE`, `CmpLT`, `CmpLTE`, `CmpEQ`, `CmpNE`
  - `type QueryType` with `QueryProbability`, `QueryExpected`, `QueryVariance`, `QueryStdDev`, `QueryDist`
- Parse errors:
  - `type ParseError` (`error`)

## Expression interface

```text
expr    = term (('+' | '-') term)*
term    = factor ('*' factor)*
factor  = repeat | atom [modifier]
repeat  = INTEGER '(' expr ')'
atom    = INTEGER
        | [INTEGER] 'd' INTEGER
        | '(' expr ')'
        | func '(' args ')'
        | 'P[' expr cmp number ']'
modifier = 'kh' INTEGER | 'kl' INTEGER | 'dh' INTEGER | 'dl' INTEGER
args    = expr (',' expr)*
func    = max | min | best | worst | adv | dis
cmp     = '>' | '>=' | '<' | '<=' | '=' | '!='
number  = INTEGER | FLOAT
```

Supported expression examples:

- `2d6 + 3`
- `d20`
- `4d6kh3`, `4d6dl1`, `4d6kl3`, `4d6dh1`
- `3 * (1d4 + 1)`
- `3(max(3, 1d6 + 1))` (three independent draws of inner expression, summed)
- `max(1, 1d4-2)`, `min(20, 1d20+5)`
- `best(3, 1d20)`, `worst(2, 1d20)`, `adv(1d20)`, `dis(1d20)`
- `P[1d20 + 7 >= 15] * (2d6 + 4)` (probability gate inside expressions)

## Query interface

```text
E[expr]         expected value
Var[expr]       variance
StdDev[expr]    standard deviation
P[expr cmp n]   probability query
D[expr]         full distribution (PMF)
expr            shorthand for D[expr]
```

Examples:

- `E[2d6+3]`
- `Var[4d6kh3]`
- `StdDev[1d20]`
- `P[adv(1d20) + 5 > 15]`
- `D[2d6]`
- `2d6` (same as `D[2d6]`)

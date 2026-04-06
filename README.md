# dicestats

![Latest Semver](https://img.shields.io/github/v/tag/Simon-Hostettler/dicestats?label=version&sort=semver)

`dicestats` is a Go library for computing statistics and probability distributions of TTRPG dice expressions.

## Usage

```go
import "hostettler.dev/dicestats"

// Query parses a query string and returns the result.
// Supports E[...], Var[...], StdDev[...], P[...], D[...],
// or a bare expression (treated as D[...]).
result, err := dicestats.Query("E[4d6kh3]")
fmt.Println(result.Value)

result, err = dicestats.Query("P[1d20 + 5 >= 15]")
fmt.Println(result.Value)

result, err = dicestats.Query("2d6 + 3")
fmt.Println(result.Distribution.Expected())
fmt.Println(result.Distribution.PMF())

// Share a cache across calls to reuse computation.
cache := dicestats.NewCache()
dicestats.Query("E[4d6kh3]", dicestats.WithCache(cache))
dicestats.Query("P[4d6kh3 >= 16]", dicestats.WithCache(cache))
```

Large expression spaces automatically fall back to Monte Carlo simulation. Use `WithSimulationThreshold`, `WithSimulationSamples`, and `WithSimulationSeed` to control this behavior. Check `result.Approximate` to see whether the result was simulated.

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

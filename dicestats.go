package dicestats

import (
	"fmt"
	"strconv"
	"strings"
)

type QueryResult struct {
	Type         QueryType
	Value        float64
	Distribution *Distribution
	Approximate  bool
}

func (r *QueryResult) String() string {
	switch r.Type {
	case QueryExpected, QueryVariance, QueryStdDev, QueryProbability:
		return strconv.FormatFloat(r.Value, 'g', 10, 64)
	case QueryDist:
		if r.Distribution == nil {
			return "<nil>"
		}
		var b strings.Builder
		for i, k := range r.Distribution.keys {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%d: %.4f", k, r.Distribution.pmf[k])
		}
		return b.String()
	default:
		return ""
	}
}

func (q QueryType) String() string {
	switch q {
	case QueryExpected:
		return "E"
	case QueryVariance:
		return "Var"
	case QueryStdDev:
		return "StdDev"
	case QueryDist:
		return "D"
	case QueryProbability:
		return "P"
	default:
		return "?"
	}
}

func Query(input string, opts ...Option) (*QueryResult, error) {
	q, err := parseQuery(input)
	if err != nil {
		return nil, err
	}

	d, err := eval(q.Expr, opts...)
	if err != nil {
		return nil, err
	}

	res := &QueryResult{Type: q.Type, Approximate: d.Approximate()}

	switch q.Type {
	case QueryExpected:
		res.Value = d.Expected()
	case QueryVariance:
		res.Value = d.Variance()
	case QueryStdDev:
		res.Value = d.StdDev()
	case QueryProbability:
		res.Value = d.PMF()[1]
	case QueryDist:
		res.Distribution = d
	}

	return res, nil
}

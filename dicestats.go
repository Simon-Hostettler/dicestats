package dicestats

type QueryResult struct {
	Type         QueryType
	Value        float64
	Distribution *Distribution
	Approximate  bool
}

func Query(input string, opts ...Option) (*QueryResult, error) {
	q, err := parseQuery(input)
	if err != nil {
		return nil, err
	}

	switch qq := q.(type) {
	case *probQuery:
		d, err := eval(qq.expr, opts...)
		if err != nil {
			return nil, err
		}
		return &QueryResult{Type: QueryProbability, Value: d.Prob(qq.Cmp, qq.Value), Approximate: d.Approximate()}, nil
	case *statQuery:
		d, err := eval(qq.expr, opts...)
		if err != nil {
			return nil, err
		}
		res := &QueryResult{Type: qq.Type, Approximate: d.Approximate()}
		switch qq.Type {
		case QueryExpected:
			res.Value = d.Expected()
		case QueryVariance:
			res.Value = d.Variance()
		case QueryStdDev:
			res.Value = d.StdDev()
		}
		return res, nil
	case *distQuery:
		d, err := eval(qq.expr, opts...)
		if err != nil {
			return nil, err
		}
		return &QueryResult{Type: QueryDist, Distribution: d, Approximate: d.Approximate()}, nil
	default:
		return nil, &ParseError{Pos: 0, Message: "unknown query type"}
	}
}

package dicestats

func parseQuery(input string) (*parsedQuery, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, &ParseError{Pos: 0, Message: "expected query"}
	}

	if tokens[0].Kind == tokenIdent {
		statPrefixes := map[string]QueryType{
			"E":      QueryExpected,
			"Var":    QueryVariance,
			"StdDev": QueryStdDev,
			"D":      QueryDist,
		}
		if qt, ok := statPrefixes[tokens[0].Text]; ok {
			e, err := parseBracketExpr(tokens, tokens[0].Text)
			if err != nil {
				return nil, err
			}
			return &parsedQuery{Type: qt, Expr: e}, nil
		}

		if tokens[0].Text == "P" {
			return parseProbQuery(tokens)
		}
	}

	e, err := parse(input)
	if err != nil {
		return nil, err
	}
	return &parsedQuery{Type: QueryDist, Expr: e}, nil
}

func parseProbQuery(tokens []token) (*parsedQuery, error) {
	p, err := parseBracketContents(tokens, "P")
	if err != nil {
		return nil, err
	}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	cmp, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	value, err := p.parseNumber()
	if err != nil {
		return nil, err
	}
	if (cmp == CmpEQ || cmp == CmpNE) && value != float64(int(value)) {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "non-integer value in equality comparison; outcomes are always integers"}
	}
	if !p.eof() {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "unexpected trailing input in P[] query"}
	}
	return &parsedQuery{Type: QueryProbability, Expr: &indicatorExpr{Inner: e, Cmp: cmp, Value: value}}, nil
}

// parseBracketContents validates PREFIX [ ... ] structure and returns a parser
// positioned at the first token inside the brackets.
func parseBracketContents(tokens []token, prefix string) (*parser, error) {
	if len(tokens) < 4 {
		return nil, &ParseError{Pos: 0, Message: "malformed " + prefix + " query"}
	}
	if tokens[0].Kind != tokenIdent || tokens[0].Text != prefix {
		return nil, &ParseError{Pos: tokens[0].Pos, Message: "malformed " + prefix + " query"}
	}
	if tokens[1].Kind != tokenSymbol || tokens[1].Text != "[" {
		return nil, &ParseError{Pos: tokens[1].Pos, Message: "malformed " + prefix + " query"}
	}
	if tokens[len(tokens)-2].Kind != tokenSymbol || tokens[len(tokens)-2].Text != "]" || tokens[len(tokens)-1].Kind != tokenEOF {
		return nil, &ParseError{Pos: tokens[len(tokens)-1].Pos, Message: "malformed " + prefix + " query"}
	}

	inside := tokens[2 : len(tokens)-2]
	inside = append(inside, token{Kind: tokenEOF, Pos: tokens[len(tokens)-2].Pos})
	return &parser{tokens: inside}, nil
}

func parseBracketExpr(tokens []token, prefix string) (expr, error) {
	p, err := parseBracketContents(tokens, prefix)
	if err != nil {
		return nil, err
	}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if !p.eof() {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "unexpected trailing input"}
	}
	return e, nil
}

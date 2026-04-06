package dicestats

func parseQuery(input string) (queryExpr, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, &ParseError{Pos: 0, Message: "expected query"}
	}

	if tokens[0].Kind == tokenIdent {
		switch tokens[0].Text {
		case "E":
			e, err := parseBracketExprTokens(tokens, "E")
			if err != nil {
				return nil, err
			}
			return &statQuery{Type: QueryExpected, expr: e}, nil
		case "Var":
			e, err := parseBracketExprTokens(tokens, "Var")
			if err != nil {
				return nil, err
			}
			return &statQuery{Type: QueryVariance, expr: e}, nil
		case "StdDev":
			e, err := parseBracketExprTokens(tokens, "StdDev")
			if err != nil {
				return nil, err
			}
			return &statQuery{Type: QueryStdDev, expr: e}, nil
		case "D":
			e, err := parseBracketExprTokens(tokens, "D")
			if err != nil {
				return nil, err
			}
			return &distQuery{expr: e}, nil
		case "P":
			pq, err := parseBracketProbTokens(tokens)
			if err != nil {
				return nil, err
			}
			return pq, nil
		}
	}

	e, err := parse(input)
	if err != nil {
		return nil, err
	}
	return &distQuery{expr: e}, nil
}

func parseBracketExprTokens(tokens []token, prefix string) (expr, error) {
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
	p := &parser{tokens: inside}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if !p.eof() {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "unexpected trailing input"}
	}
	return expr, nil
}

func parseBracketProbTokens(tokens []token) (*probQuery, error) {
	if len(tokens) < 7 {
		return nil, &ParseError{Pos: 0, Message: "malformed P query"}
	}
	if tokens[0].Kind != tokenIdent || tokens[0].Text != "P" {
		return nil, &ParseError{Pos: tokens[0].Pos, Message: "malformed P query"}
	}
	if tokens[1].Kind != tokenSymbol || tokens[1].Text != "[" {
		return nil, &ParseError{Pos: tokens[1].Pos, Message: "malformed P query"}
	}
	if tokens[len(tokens)-2].Kind != tokenSymbol || tokens[len(tokens)-2].Text != "]" || tokens[len(tokens)-1].Kind != tokenEOF {
		return nil, &ParseError{Pos: tokens[len(tokens)-1].Pos, Message: "malformed P query"}
	}

	inside := tokens[2 : len(tokens)-2]
	inside = append(inside, token{Kind: tokenEOF, Pos: tokens[len(tokens)-2].Pos})
	p := &parser{tokens: inside}

	expr, err := p.parseExpr()
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
	if !p.eof() {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "unexpected trailing input"}
	}
	return &probQuery{expr: expr, Cmp: cmp, Value: value}, nil
}

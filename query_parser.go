package dicestats

func parseQuery(input string) (*queryExpr, error) {
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
			return &queryExpr{Type: qt, Expr: e}, nil
		}

		// P[expr cmp value] → QueryDist wrapping a probExpr.
		// Only when the entire input is P[...] (bracket is last token).
		// Otherwise fall through to parse as a bare expression containing
		// a probExpr sub-expression (e.g. "P[1d20+5>15] * 2d6").
		if tokens[0].Text == "P" {
			if p, err := parseBracketContents(tokens, "P"); err == nil {
				e, err := p.parseExpr()
				if err == nil {
					cmp, err := p.parseCmp()
					if err == nil {
						value, err := p.parseNumber()
						if err == nil && p.eof() {
							return &queryExpr{Type: QueryDist, Expr: &probExpr{Inner: e, Cmp: cmp, Value: value}}, nil
						}
					}
				}
			}
		}
	}

	e, err := parse(input)
	if err != nil {
		return nil, err
	}
	return &queryExpr{Type: QueryDist, Expr: e}, nil
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

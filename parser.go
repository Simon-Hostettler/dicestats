package dicestats

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	tokens []token
	pos    int
}

func parse(input string) (expr, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if !p.eof() {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "unexpected trailing input"}
	}
	return expr, nil
}

func (p *parser) parseExpr() (expr, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for {
		if p.matchSymbol("+") {
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			left = &binaryExpr{Op: opAdd, Left: left, Right: right}
			continue
		}
		if p.matchSymbol("-") {
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			left = &binaryExpr{Op: opSub, Left: left, Right: right}
			continue
		}
		break
	}
	return left, nil
}

func (p *parser) parseTerm() (expr, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for {
		if !p.matchSymbol("*") {
			break
		}
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &binaryExpr{Op: opMul, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseFactor() (expr, error) {
	if p.peek().Kind == tokenInt && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind == tokenSymbol && p.tokens[p.pos+1].Text == "(" {
		start := p.peek().Pos
		n, err := p.parseIntLiteral()
		if err != nil {
			return nil, err
		}
		if n <= 0 {
			return nil, &ParseError{Pos: start, Message: "repeat count must be > 0"}
		}
		if !p.matchSymbol("(") {
			return nil, &ParseError{Pos: p.peek().Pos, Message: "expected '('"}
		}
		base, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if !p.matchSymbol(")") {
			return nil, &ParseError{Pos: p.peek().Pos, Message: "expected ')'"}
		}
		kind, _, hasModifier, err := p.parseKeepDropModifier()
		if err != nil {
			return nil, err
		}
		if hasModifier {
			return nil, &ParseError{Pos: p.peek().Pos, Message: kind.String() + " modifiers are only allowed on dice literals"}
		}
		return &repeatExpr{Count: n, Base: base}, nil
	}

	atom, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	kind, n, hasModifier, err := p.parseKeepDropModifier()
	if err != nil {
		return nil, err
	}
	if !hasModifier {
		return atom, nil
	}
	if _, ok := atom.(*diceExpr); !ok {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "keep/drop modifiers are only allowed on dice literals"}
	}
	atom = &keepDropExpr{Base: atom, Kind: kind, N: n}
	if _, _, again, err := p.parseKeepDropModifier(); err != nil {
		return nil, err
	} else if again {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "multiple keep/drop modifiers are not allowed"}
	}
	return atom, nil
}

func (p *parser) parseKeepDropModifier() (keepDropKind, int, bool, error) {
	type candidate struct {
		token string
		kind  keepDropKind
	}
	candidates := []candidate{
		{token: "kh", kind: keepHighest},
		{token: "kl", kind: keepLowest},
		{token: "dh", kind: dropHighest},
		{token: "dl", kind: dropLowest},
	}
	for _, c := range candidates {
		if p.matchIdent(c.token) {
			n, err := p.parseIntLiteral()
			if err != nil {
				return 0, 0, true, err
			}
			return c.kind, n, true, nil
		}
	}
	return 0, 0, false, nil
}

func (p *parser) parseAtom() (expr, error) {
	if p.eof() {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "expected expression"}
	}

	if p.matchSymbol("(") {
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if !p.matchSymbol(")") {
			return nil, &ParseError{Pos: p.peek().Pos, Message: "expected ')'"}
		}
		return e, nil
	}

	if p.isProbAtomStart() {
		return p.parseProbAtom()
	}

	if p.peek().Kind == tokenInt {
		start := p.peek().Pos
		n, err := p.parseIntLiteral()
		if err != nil {
			return nil, err
		}
		if p.matchIdent("d") {
			sides, err := p.parseIntLiteral()
			if err != nil {
				return nil, err
			}
			if n <= 0 {
				return nil, &ParseError{Pos: start, Message: "dice count must be > 0"}
			}
			if sides <= 0 {
				return nil, &ParseError{Pos: p.peek().Pos, Message: "dice sides must be > 0"}
			}
			return &diceExpr{Count: n, Sides: sides}, nil
		}
		return &numberExpr{Value: n}, nil
	}

	if p.matchIdent("d") {
		sides, err := p.parseIntLiteral()
		if err != nil {
			return nil, err
		}
		if sides <= 0 {
			return nil, &ParseError{Pos: p.peek().Pos, Message: "dice sides must be > 0"}
		}
		return &diceExpr{Count: 1, Sides: sides}, nil
	}

	if p.peek().Kind == tokenIdent {
		return p.parseFuncCall()
	}

	return nil, &ParseError{Pos: p.peek().Pos, Message: fmt.Sprintf("unexpected character '%s'", p.peek().Text)}
}

func (p *parser) parseProbAtom() (expr, error) {
	if !p.matchIdent("P") || !p.matchSymbol("[") {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "expected 'P['"}
	}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	cmp, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	val, err := p.parseNumber()
	if err != nil {
		return nil, err
	}
	if !p.matchSymbol("]") {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "expected ']'"}
	}
	return &probExpr{expr: e, Cmp: cmp, Value: val}, nil
}

func (p *parser) isProbAtomStart() bool {
	if p.peek().Kind != tokenIdent || p.peek().Text != "P" {
		return false
	}
	if p.pos+1 >= len(p.tokens) {
		return false
	}
	next := p.tokens[p.pos+1]
	return next.Kind == tokenSymbol && next.Text == "["
}

func (p *parser) parseFuncCall() (expr, error) {
	start := p.peek().Pos
	name := p.peek().Text
	p.next()
	lname := strings.ToLower(name)
	arity, ok := functionArity(lname)
	if !ok {
		return nil, &ParseError{Pos: start, Message: "unknown function: " + name}
	}
	if !p.matchSymbol("(") {
		return nil, &ParseError{Pos: p.peek().Pos, Message: "expected '(' after function name"}
	}
	args := make([]expr, 0, 2)
	if p.matchSymbol(")") {
		return &funcExpr{Name: lname, Args: args}, nil
	}
	for {
		a, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, a)
		if p.matchSymbol(")") {
			break
		}
		if !p.matchSymbol(",") {
			return nil, &ParseError{Pos: p.peek().Pos, Message: "expected ',' or ')'"}
		}
	}
	if len(args) != arity {
		return nil, &ParseError{Pos: start, Message: fmt.Sprintf("%s expects %d args", lname, arity)}
	}
	return &funcExpr{Name: lname, Args: args}, nil
}

func (p *parser) parseCmp() (Cmp, error) {
	if p.matchSymbol(">") {
		if p.matchSymbol("=") {
			return CmpGTE, nil
		}
		return CmpGT, nil
	}
	if p.matchSymbol("<") {
		if p.matchSymbol("=") {
			return CmpLTE, nil
		}
		return CmpLT, nil
	}
	if p.matchSymbol("!") {
		if p.matchSymbol("=") {
			return CmpNE, nil
		}
		return 0, &ParseError{Pos: p.peek().Pos, Message: "expected comparison operator"}
	}
	if p.matchSymbol("=") {
		return CmpEQ, nil
	}
	return 0, &ParseError{Pos: p.peek().Pos, Message: "expected comparison operator"}
}

func (p *parser) parseNumber() (float64, error) {
	tok := p.peek()
	switch tok.Kind {
	case tokenInt, tokenFloat:
		p.next()
		v, err := strconv.ParseFloat(tok.Text, 64)
		if err != nil {
			return 0, &ParseError{Pos: tok.Pos, Message: "invalid number"}
		}
		return v, nil
	default:
		return 0, &ParseError{Pos: tok.Pos, Message: "expected number"}
	}
}

func (p *parser) parseIntLiteral() (int, error) {
	tok := p.peek()
	if tok.Kind != tokenInt {
		return 0, &ParseError{Pos: tok.Pos, Message: "expected integer"}
	}
	v, err := strconv.Atoi(tok.Text)
	if err != nil {
		return 0, &ParseError{Pos: tok.Pos, Message: "invalid integer"}
	}
	p.next()
	return v, nil
}

func (p *parser) next() {
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
}

func (p *parser) matchIdent(name string) bool {
	tok := p.peek()
	if tok.Kind == tokenIdent && tok.Text == name {
		p.next()
		return true
	}
	return false
}

func (p *parser) matchSymbol(symbol string) bool {
	tok := p.peek()
	if tok.Kind == tokenSymbol && tok.Text == symbol {
		p.next()
		return true
	}
	return false
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{Kind: tokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) eof() bool {
	return p.peek().Kind == tokenEOF
}

package dicestats

import (
	"unicode"
)

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenInt
	tokenFloat
	tokenIdent
	tokenSymbol
)

type token struct {
	Kind tokenKind
	Text string
	Pos  int
}

func tokenize(input string) ([]token, error) {
	tokens := make([]token, 0, len(input)/2+1)
	for pos := 0; pos < len(input); {
		ch := input[pos]
		if unicode.IsSpace(rune(ch)) {
			pos++
			continue
		}

		if isDigit(ch) {
			start := pos
			hasDot := false
			for pos < len(input) {
				c := input[pos]
				if isDigit(c) {
					pos++
					continue
				}
				if c == '.' {
					if hasDot {
						break
					}
					hasDot = true
					pos++
					continue
				}
				break
			}
			kind := tokenInt
			if hasDot {
				kind = tokenFloat
			}
			tokens = append(tokens, token{Kind: kind, Text: input[start:pos], Pos: start})
			continue
		}

		if isLetter(ch) {
			start := pos
			for pos < len(input) && isLetter(input[pos]) {
				pos++
			}
			tokens = append(tokens, token{Kind: tokenIdent, Text: input[start:pos], Pos: start})
			continue
		}

		switch ch {
		case '+', '-', '*', '(', ')', '[', ']', ',', '>', '<', '!', '=':
			tokens = append(tokens, token{Kind: tokenSymbol, Text: string(ch), Pos: pos})
			pos++
		default:
			return nil, &ParseError{Pos: pos, Message: "unexpected character '" + string(ch) + "'"}
		}
	}
	tokens = append(tokens, token{Kind: tokenEOF, Pos: len(input)})
	return tokens, nil
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

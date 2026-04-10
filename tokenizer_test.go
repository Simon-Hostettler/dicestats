package dicestats

import "testing"

func TestTokenizeSkipsWhitespace(t *testing.T) {
	tokens, err := tokenize(" \t2d6 + 1 \n")
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	wantKinds := []tokenKind{tokenInt, tokenIdent, tokenInt, tokenSymbol, tokenInt, tokenEOF}
	wantText := []string{"2", "d", "6", "+", "1", ""}
	if len(tokens) != len(wantKinds) {
		t.Fatalf("unexpected token count: got %d want %d", len(tokens), len(wantKinds))
	}
	for i := range tokens {
		if tokens[i].Kind != wantKinds[i] || tokens[i].Text != wantText[i] {
			t.Fatalf("token %d mismatch: got (%v,%q) want (%v,%q)", i, tokens[i].Kind, tokens[i].Text, wantKinds[i], wantText[i])
		}
	}
}

func TestTokenizeRecognizesFloatForProbQuery(t *testing.T) {
	tokens, err := tokenize("P[2d6 >= 10.5]")
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	foundFloat := false
	for _, tok := range tokens {
		if tok.Kind == tokenFloat && tok.Text == "10.5" {
			foundFloat = true
			break
		}
	}
	if !foundFloat {
		t.Fatal("expected float token 10.5")
	}
}

func TestParseWithWhitespaceEverywhere(t *testing.T) {
	_, err := parse(" 4 d6 \t kh 3 + [ 1 d20 + 5 >= 15 ] * 2 ")
	if err != nil {
		t.Fatalf("expected parser to accept whitespace-heavy input, got: %v", err)
	}
}

func TestTokenizeUnexpectedCharacter(t *testing.T) {
	_, err := tokenize("2d6$1")
	if err == nil {
		t.Fatal("expected tokenizer error")
	}
}

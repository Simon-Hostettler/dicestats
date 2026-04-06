package dicestats

import (
	"fmt"
	"strings"
)

type functionKind int

const (
	functionMax functionKind = iota
	functionMin
	functionBest
	functionWorst
	functionAdv
	functionDis
)

type functionCall struct {
	kind   functionKind
	first  expr
	second expr
	n      int
}

func functionArity(name string) (int, bool) {
	switch strings.ToLower(name) {
	case "max", "min", "best", "worst":
		return 2, true
	case "adv", "dis":
		return 1, true
	default:
		return 0, false
	}
}

func parseFunctionCall(e *funcExpr) (functionCall, error) {
	name := strings.ToLower(e.Name)
	arity, ok := functionArity(name)
	if !ok {
		return functionCall{}, fmt.Errorf("unsupported function %s", e.Name)
	}
	if len(e.Args) != arity {
		return functionCall{}, fmt.Errorf("%s expects %d args", name, arity)
	}
	switch name {
	case "max":
		return functionCall{kind: functionMax, first: e.Args[0], second: e.Args[1]}, nil
	case "min":
		return functionCall{kind: functionMin, first: e.Args[0], second: e.Args[1]}, nil
	case "adv":
		return functionCall{kind: functionAdv, first: e.Args[0], n: 2}, nil
	case "dis":
		return functionCall{kind: functionDis, first: e.Args[0], n: 2}, nil
	case "best", "worst":
		nExpr, ok := e.Args[0].(*numberExpr)
		if !ok || nExpr.Value <= 0 {
			return functionCall{}, fmt.Errorf("%s first arg must be positive integer literal", name)
		}
		kind := functionBest
		if name == "worst" {
			kind = functionWorst
		}
		return functionCall{kind: kind, first: e.Args[1], n: nExpr.Value}, nil
	default:
		return functionCall{}, fmt.Errorf("unsupported function %s", e.Name)
	}
}

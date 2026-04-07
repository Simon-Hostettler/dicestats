package dicestats

import (
	"fmt"
	"math/rand"
	"strconv"
)

type expr interface {
	Key() string
	eval(cfg *config) (*Distribution, error)
	sample(rng *rand.Rand, cfg *config) (int, error)
	estimateCost() int
}

type Cmp int

const (
	CmpGT Cmp = iota
	CmpGTE
	CmpLT
	CmpLTE
	CmpEQ
	CmpNE
)

func (c Cmp) String() string {
	switch c {
	case CmpGT:
		return ">"
	case CmpGTE:
		return ">="
	case CmpLT:
		return "<"
	case CmpLTE:
		return "<="
	case CmpEQ:
		return "="
	case CmpNE:
		return "!="
	default:
		return "?"
	}
}

type numberExpr struct {
	Value int
}

func (n *numberExpr) Key() string {
	return fmt.Sprintf("n(%d)", n.Value)
}

type diceExpr struct {
	Count int
	Sides int
}

func (d *diceExpr) Key() string {
	return fmt.Sprintf("d(%d,%d)", d.Count, d.Sides)
}

type repeatExpr struct {
	Count int
	Base  expr
}

func (r *repeatExpr) Key() string {
	return fmt.Sprintf("rep(%d,%s)", r.Count, r.Base.Key())
}

type binaryOp int

const (
	opAdd binaryOp = iota
	opSub
	opMul
)

func (o binaryOp) String() string {
	switch o {
	case opAdd:
		return "+"
	case opSub:
		return "-"
	case opMul:
		return "*"
	default:
		return "?"
	}
}

type binaryExpr struct {
	Op    binaryOp
	Left  expr
	Right expr
}

func (b *binaryExpr) Key() string {
	return "(" + b.Op.String() + "," + b.Left.Key() + "," + b.Right.Key() + ")"
}

type keepDropKind int

const (
	keepHighest keepDropKind = iota
	keepLowest
	dropHighest
	dropLowest
)

func (k keepDropKind) String() string {
	switch k {
	case keepHighest:
		return "kh"
	case keepLowest:
		return "kl"
	case dropHighest:
		return "dh"
	case dropLowest:
		return "dl"
	default:
		return "?"
	}
}

type keepDropExpr struct {
	Base expr
	Kind keepDropKind
	N    int
}

func (k *keepDropExpr) Key() string {
	return k.Kind.String() + "(" + k.Base.Key() + "," + strconv.Itoa(k.N) + ")"
}

type functionKind int

const (
	functionMax functionKind = iota
	functionMin
	functionBest
	functionWorst
	functionAdv
	functionDis
)

var functionDefs = map[string]struct {
	arity int
	kind  functionKind
}{
	"max": {2, functionMax}, "min": {2, functionMin},
	"best": {2, functionBest}, "worst": {2, functionWorst},
	"adv": {1, functionAdv}, "dis": {1, functionDis},
}

// binaryFuncExpr represents binary comparison functions (max, min).
type binaryFuncExpr struct {
	Kind  functionKind
	Name  string
	Left  expr
	Right expr
}

func (f *binaryFuncExpr) Key() string {
	return f.Name + "(" + f.Left.Key() + "," + f.Right.Key() + ")"
}

// orderStatExpr represents order-statistic functions (best, worst, adv, dis).
type orderStatExpr struct {
	Kind functionKind
	Name string
	Base expr
	N    int
}

func (f *orderStatExpr) Key() string {
	return f.Name + "(" + strconv.Itoa(f.N) + "," + f.Base.Key() + ")"
}

type probExpr struct {
	Inner expr
	Cmp   Cmp
	Value float64
}

func (p *probExpr) Key() string {
	return "P[" + p.Inner.Key() + p.Cmp.String() + strconv.FormatFloat(p.Value, 'g', -1, 64) + "]"
}

type QueryType int

const (
	QueryProbability QueryType = iota
	QueryExpected
	QueryVariance
	QueryStdDev
	QueryDist
)

type parsedQuery struct {
	Type QueryType
	Expr expr
}

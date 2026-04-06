package dicestats

import (
	"fmt"
	"strconv"
	"strings"
)

type expr interface {
	exprNode()
	Key() string
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

func (*numberExpr) exprNode() {}
func (n *numberExpr) Key() string {
	return fmt.Sprintf("n(%d)", n.Value)
}

type diceExpr struct {
	Count int
	Sides int
}

func (*diceExpr) exprNode() {}
func (d *diceExpr) Key() string {
	return fmt.Sprintf("d(%d,%d)", d.Count, d.Sides)
}

type repeatExpr struct {
	Count int
	Base  expr
}

func (*repeatExpr) exprNode() {}
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

func (*binaryExpr) exprNode() {}
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

func (*keepDropExpr) exprNode() {}
func (k *keepDropExpr) Key() string {
	return k.Kind.String() + "(" + k.Base.Key() + "," + strconv.Itoa(k.N) + ")"
}

type funcExpr struct {
	Name string
	Args []expr
}

func (*funcExpr) exprNode() {}
func (f *funcExpr) Key() string {
	parts := make([]string, 0, len(f.Args))
	for _, a := range f.Args {
		parts = append(parts, a.Key())
	}
	return strings.ToLower(f.Name) + "(" + strings.Join(parts, ",") + ")"
}

type probExpr struct {
	expr  expr
	Cmp   Cmp
	Value float64
}

func (*probExpr) exprNode() {}
func (p *probExpr) Key() string {
	return "P[" + p.expr.Key() + p.Cmp.String() + strconv.FormatFloat(p.Value, 'g', -1, 64) + "]"
}

type QueryType int

const (
	QueryProbability QueryType = iota
	QueryExpected
	QueryVariance
	QueryStdDev
	QueryDist
)

type queryExpr interface {
	queryNode()
}

type statQuery struct {
	Type QueryType
	expr expr
}

func (*statQuery) queryNode() {}

type probQuery struct {
	expr  expr
	Cmp   Cmp
	Value float64
}

func (*probQuery) queryNode() {}

type distQuery struct {
	expr expr
}

func (*distQuery) queryNode() {}

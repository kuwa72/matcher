// nolint: govet
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alecthomas/kong"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/participle/v2/lexer/stateful"

	"github.com/alecthomas/repr"
)

type Boolean bool

type Context map[string]string

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "TRUE"
	return nil
}

type Expression struct {
	Or []*OrCondition `@@ ( "OR" @@ )*`
}

func (e *Expression) Eval(ctx Context) (bool, error) {
	for _, x := range e.Or {
		if b, err := x.Eval(ctx); err != nil {
			return false, err
		} else if b {
			return true, nil
		}
	}
	return false, nil
}

type OrCondition struct {
	And []*Condition `@@ ( "AND" @@ )*`
}

func (e *OrCondition) Eval(ctx Context) (bool, error) {
	for _, x := range e.And {
		if b, err := x.Eval(ctx); err != nil {
			return false, err
		} else if !b {
			return false, nil
		}
	}
	return true, nil
}

type Condition struct {
	Operand *ConditionOperand `  @@`
}

func (x *Condition) Eval(ctx Context) (bool, error) {
	return x.Operand.Eval(ctx)
}

type ConditionOperand struct {
	Operand      *Operand      `@@`
	ConditionRHS *ConditionRHS `@@?`
}

func (x *ConditionOperand) Eval(ctx Context) (bool, error) {
	sym := x.Operand.Summand[0].LHS.LHS.SymbolRef.Symbol
	ctxVal, ok := ctx[sym]
	if !ok {
		return false, nil
	}

	switch o := x.ConditionRHS.Compare.Operator; o {
	case "=":
		v := x.ConditionRHS.Compare.Operand.Summand[0].LHS.LHS.Value
		fmt.Printf("%#v = %#v\n", ctxVal, v)
		switch {
		case v.Number != nil:
			return ctxVal == fmt.Sprintf("%d", *v.Number), nil
		case v.String != nil:
			return ctxVal == *v.String, nil
		default:
			return false, fmt.Errorf("Unknown value type: %#v", v)
		}
	default:
		return false, fmt.Errorf("Unknown operator: %s", o)
	}
}

type ConditionRHS struct {
	Compare *Compare `  @@`
}

type Compare struct {
	Operator string   `@( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	Operand  *Operand `(  @@ )`
}

type Operand struct {
	Summand []*Summand `@@ ( "|" "|" @@ )*`
}

type Summand struct {
	LHS *Factor `@@`
	Op  string  `[ @("+" | "-")`
	RHS *Factor `  @@ ]`
}

type Factor struct {
	LHS *Term  `@@`
	Op  string `( @("*" | "/" | "%")`
	RHS *Term  `  @@ )?`
}

type Term struct {
	Value     *Value     `@@`
	SymbolRef *SymbolRef `| @@`
}

type SymbolRef struct {
	Symbol     string        `@Ident @( "." Ident )*`
	Parameters []*Expression `( "(" @@ ( "," @@ )* ")" )?`
}

type Value struct {
	Number  *float64 `( @Number`
	String  *string  ` | @String`
	Boolean *Boolean ` | @("TRUE" | "FALSE")`
	Null    bool     ` | @"NULL"`
	Array   *Array   ` | @@ )`
}

type Array struct {
	Expressions []*Expression `"(" @@ ( "," @@ )* ")"`
}

var (
	cli struct {
		QUERY string `arg:"" required:"" help:"QUERY to parse."`
	}

	qLexer = lexer.Must(stateful.NewSimple([]stateful.Rule{
		{`Keyword`, `(?i)TRUE|FALSE|BETWEEN|AND|OR|LIKE|IN`, nil},
		{`Ident`, `[a-zA-Z_][a-zA-Z0-9_]*`, nil},
		{`Number`, `[-+]?\d*\.?\d+([eE][-+]?\d+)?`, nil},
		{`String`, `'[^']*'|"[^"]*"`, nil},
		{`Operators`, `<>|!=|<=|>=|[-+*/%,.()=<>]`, nil},
		{"whitespace", `\s+`, nil},
	}))
	parser = participle.MustBuild(
		&Expression{},
		participle.Lexer(qLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Keyword"),
		// participle.Elide("Comment"),
		// Need to solve left recursion detection first, if possible.
		// participle.UseLookahead(),
	)
)

func main() {
	ctx := kong.Parse(&cli)
	e := &Expression{}
	err := parser.ParseString("", cli.QUERY, e)
	repr.Println(e, repr.Indent("  "), repr.OmitEmpty(true))
	ctx.FatalIfErrorf(err)

	j, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c := Context(make(map[string]string))
	json.Unmarshal([]byte(j), &c)

	b, err := e.Eval(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	switch {
	case b:
		fmt.Println("matched")
	default:
		fmt.Println("Unmatched")
	}
}

package matcher

import (
	"fmt"
	"strconv"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/participle/v2/lexer/stateful"
)

type Boolean bool

type Context map[string]interface{}

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
	Symbol  string   `@Ident`
	Compare *Compare `@@`
}

func (x *Condition) Eval(ctx Context) (bool, error) {
	sym := x.Symbol
	ctxVal, ok := ctx[sym]
	if !ok {
		return false, nil
	}

	switch o := x.Compare.Operator; o {
	case "=":
		v := x.Compare.Value
		switch {
		case v.Number != nil:
			switch x := ctxVal.(type) {
			case float64:
				return int(x) == *v.Number, nil
			case int:
				return x == *v.Number, nil
			case string:
				return x == strconv.Itoa(*v.Number), nil
			case bool:
				return x && *v.Number != 0 || !x && *v.Number == 0, nil // 0 is false, otherwise true
			}
		case v.String != nil:
			return ctxVal == *v.String, nil
		case v.Boolean != nil:
			switch x := ctxVal.(type) {
			case int:
				return x == 0 && !(*v.Boolean) || x != 0 && (*v.Boolean), nil // 0 is false, otherwise true
			case bool:
				return x == *v.Boolean, nil
			case string:
				b, err := strconv.ParseBool(x)
				if err != nil {
					return false, fmt.Errorf("Is not bool value:%s, %w", x, err)
				}
				return b == *v.Boolean, nil
			}
		default:
			return false, fmt.Errorf("Unknown value type: %#v", v)
		}
	default:
		return false, fmt.Errorf("Unknown operator: %s", o)
	}
	return false, fmt.Errorf("Failed to complation, type: %T: %#v", ctxVal, ctxVal)
}

type Compare struct {
	Operator string `@( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	Value    *Value `@@`
}

type Value struct {
	Number  *int    `( @Number`
	String  *string ` | @String`
	Boolean *bool   ` | @("TRUE" | "FALSE")`
	Null    bool    ` | @"NULL" )`
}

func NewParser() *participle.Parser {
	qLexer := lexer.Must(stateful.NewSimple([]stateful.Rule{
		{`Keyword`, `(?i)TRUE|FALSE|AND|OR`, nil},
		{`Ident`, `[a-zA-Z_][a-zA-Z0-9_]*`, nil},
		{`Number`, `[-+]?\d*\.?\d+([eE][-+]?\d+)?`, nil},
		{`String`, `'[^']*'|"[^"]*"`, nil},
		{`Operators`, `<>|!=|<=|>=|[-+*/%,.()=<>]`, nil},
		{"whitespace", `\s+`, nil},
	}))
	return participle.MustBuild(
		&Expression{},
		participle.Lexer(qLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Keyword"),
		// participle.Elide("Comment"),
		// Need to solve left recursion detection first, if possible.
		// participle.UseLookahead(),
	)
}

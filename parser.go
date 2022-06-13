package matcher

import (
	"fmt"
	"strconv"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
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
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				return x == *v.Float, nil
			case string:
				return x == fmt.Sprintf("%f", *v.Float), nil
			case bool:
				return x && *v.Float != 0 || !x && *v.Float == 0, nil // 0 is false, otherwise true
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
					return false, fmt.Errorf("is not bool value:%s, %w", x, err)
				}
				return b == *v.Boolean, nil
			}
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}
	case "<>", "!=":
		v := x.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				return x != *v.Float, nil
			case string:
				return x != fmt.Sprintf("%f", *v.Float), nil
			case bool:
				return !(x && *v.Float != 0 || !x && *v.Float == 0), nil // 0 is false, otherwise true
			}
		case v.String != nil:
			return ctxVal != *v.String, nil
		case v.Boolean != nil:
			switch x := ctxVal.(type) {
			case int:
				return !(x == 0 && !(*v.Boolean) || x != 0 && (*v.Boolean)), nil // 0 is false, otherwise true
			case bool:
				return x != *v.Boolean, nil
			case string:
				b, err := strconv.ParseBool(x)
				if err != nil {
					return false, fmt.Errorf("is not bool value:%s, %w", x, err)
				}
				return b != *v.Boolean, nil
			}
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case ">":
		v := x.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64:
				return x.(float64) > *v.Float, nil
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				i := x.(int64)
				return float64(i) > *v.Float, nil
			case string:
				return string(x) > fmt.Sprintf("%f", *v.Float), nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
			}
		case v.String != nil:
			return ctxVal.(string) > *v.String, nil
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case ">=":
		v := x.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64:
				return x.(float64) >= *v.Float, nil
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				i := x.(int64)
				return float64(i) >= *v.Float, nil
			case string:
				return string(x) >= fmt.Sprintf("%f", *v.Float), nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
			}
		case v.String != nil:
			return ctxVal.(string) >= *v.String, nil
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case "<":
		v := x.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64:
				return x.(float64) < *v.Float, nil
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				i := x.(int64)
				return float64(i) < *v.Float, nil
			case string:
				return string(x) < fmt.Sprintf("%f", *v.Float), nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
			}
		case v.String != nil:
			return ctxVal.(string) < *v.String, nil
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case "<=":
		v := x.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64:
				return x.(float64) <= *v.Float, nil
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				i := x.(int64)
				return float64(i) <= *v.Float, nil
			case string:
				return string(x) <= fmt.Sprintf("%f", *v.Float), nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
			}
		case v.String != nil:
			return ctxVal.(string) <= *v.String, nil
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less then: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	default:
		return false, fmt.Errorf("unknown operator: %s", o)
	}
	return false, fmt.Errorf("failed to complation, type: %T: %#v", ctxVal, ctxVal)
}

type Compare struct {
	Operator string `@( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	Value    *Value `@@`
}

type Value struct {
	Float   *float64 `( @Float `
	String  *string  ` | @String`
	Boolean *bool    ` | @("TRUE" | "FALSE")`
	Null    bool     ` | @"NULL" )`
}

func NewParser() *participle.Parser {
	qLexer := lexer.MustSimple([]lexer.SimpleRule{
		{`Keyword`, `(?i)TRUE|FALSE|AND|OR`},
		{`Ident`, `[a-zA-Z_][a-zA-Z0-9_]*`},
		{`Float`, `[-+]?\d*\.?\d+([eE][-+]?\d+)?`},
		{`String`, `'[^']*'|"[^"]*"`},
		{`Operators`, `<>|!=|<=|>=|[-+*/%,.()=<>]`},
		{"whitespace", `\s+`},
	})
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

package matcher

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Boolean is a custom boolean type for parsing boolean values from query strings
type Boolean bool

// Context is a map of string keys to arbitrary values that can be evaluated against expressions
type Context map[string]interface{}

// ErrInvalidContext is returned when an invalid context is provided
var ErrInvalidContext = errors.New("invalid context")

// ErrInvalidValue is returned when a value cannot be properly compared
var ErrInvalidValue = errors.New("invalid value")

// ErrInvalidOperator is returned when an unknown operator is encountered
var ErrInvalidOperator = errors.New("invalid operator")

// Capture implements the participle.Capture interface for Boolean
func (b *Boolean) Capture(values []string) error {
	if len(values) == 0 {
		return errors.New("no values to capture")
	}
	*b = values[0] == "TRUE"
	return nil
}

// Expression represents a parsed query expression with OR conditions
type Expression struct {
	Or []*OrCondition `@@ ( "OR" @@ )*`
}

// Eval evaluates the expression against the provided context
// Returns true if any of the OR conditions evaluate to true
func (e *Expression) Eval(ctx Context) (bool, error) {
	if e == nil || len(e.Or) == 0 {
		return false, nil
	}
	
	for _, x := range e.Or {
		result, err := x.Eval(ctx)
		if err != nil {
			return false, fmt.Errorf("evaluating OR condition: %w", err)
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

// OrCondition represents a set of AND conditions within an expression
type OrCondition struct {
	And []*Condition `@@ ( "AND" @@ )*`
}

// Eval evaluates the AND conditions against the provided context
// Returns true only if all AND conditions evaluate to true
func (e *OrCondition) Eval(ctx Context) (bool, error) {
	if e == nil || len(e.And) == 0 {
		return false, nil
	}
	
	for _, x := range e.And {
		result, err := x.Eval(ctx)
		if err != nil {
			return false, fmt.Errorf("evaluating AND condition: %w", err)
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

// Condition represents either a simple condition or a nested expression in parentheses
type Condition struct {
	// Only one of these will be set
	Nested    *Expression `  "(" @@ ")"`
	Predicate *Predicate  `| @@`
}

// Eval evaluates the condition against the provided context
func (x *Condition) Eval(ctx Context) (bool, error) {
	if x == nil {
		return false, errors.New("invalid condition")
	}
	
	// If this is a nested expression in parentheses, evaluate it
	if x.Nested != nil {
		return x.Nested.Eval(ctx)
	}
	
	// Otherwise evaluate the predicate
	if x.Predicate == nil {
		return false, errors.New("invalid predicate")
	}
	
	return x.Predicate.Eval(ctx)
}

// Predicate represents a simple condition with a symbol and comparison
type Predicate struct {
	Symbol  string   `@Ident`
	Compare *Compare `@@`
}

// Eval evaluates the predicate against the provided context
func (p *Predicate) Eval(ctx Context) (bool, error) {
	if p == nil || p.Compare == nil {
		return false, errors.New("invalid predicate")
	}
	
	sym := p.Symbol
	ctxVal, ok := ctx[sym]
	if !ok {
		// Symbol not found in context, return false but not an error
		return false, nil
	}

	switch o := p.Compare.Operator; o {
	case "=":
		v := p.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64:
				return x.(float64) == *v.Float, nil
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				return (float64)(x.(int)) == *v.Float, nil
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
		v := p.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64:
				return x.(float64) != *v.Float, nil
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				return (float64)(x.(int)) != *v.Float, nil
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
		v := p.Compare.Value
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
		v := p.Compare.Value
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
		v := p.Compare.Value
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
		v := p.Compare.Value
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
	return false, fmt.Errorf("failed to complete comparison, type: %T: %#v", ctxVal, ctxVal)
}

// Compare represents a comparison operation with an operator and value
type Compare struct {
	Operator string `@( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	Value    *Value `@@`
}

// Value represents a value that can be compared in a condition
type Value struct {
	Float   *float64 `( @Float `
	String  *string  ` | @String`
	Boolean *bool    ` | @("TRUE" | "FALSE")`
	Null    bool     ` | @"NULL" )`
}

// NewParser creates a new participle parser for parsing query expressions
func NewParser() *participle.Parser[Expression] {
	qLexer := lexer.MustSimple([]lexer.SimpleRule{
		{`Keyword`, `(?i)TRUE|FALSE|AND|OR|NULL`},
		{`Ident`, `[a-zA-Z_][a-zA-Z0-9_]*`},
		{`Float`, `[-+]?\d*\.?\d+([eE][-+]?\d+)?`},
		{`String`, `'[^']*'|"[^"]*"`},
		{`Operators`, `<>|!=|<=|>=|[-+*/%,.()=<>]`},
		{`Parentheses`, `[\(\)]`},
		{"whitespace", `\s+`},
	})
	return participle.MustBuild[Expression](
		participle.Lexer(qLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Keyword"),
		participle.UseLookahead(20),
	)
}

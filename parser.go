package matcher

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	Or []*OrCondition `parser:"@@ ( \"OR\" @@ )*"`
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
	And []*Condition `parser:"@@ ( \"AND\" @@ )*"`
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
	Nested    *Expression `parser:"  \"(\" @@ \")\""`
	Predicate *Predicate  `parser:"| @@"`
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
	Symbol  string   `parser:"@Ident"`
	Compare *Compare `parser:"@@"`
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
		case v.Regex != nil:
			strVal, ok := ctxVal.(string)
			if !ok {
				return false, fmt.Errorf("cannot apply regex to non-string value: %T", ctxVal)
			}
			return v.Regex.Regexp.MatchString(strVal), nil
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
		case v.Regex != nil:
			strVal, ok := ctxVal.(string)
			if !ok {
				return false, fmt.Errorf("cannot apply regex to non-string value: %T", ctxVal)
			}
			return !v.Regex.Regexp.MatchString(strVal), nil
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
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use > operator with regex pattern")
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
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use >= operator with regex pattern")
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
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use < operator with regex pattern")
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
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use <= operator with regex pattern")
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
	Operator string `parser:"@( \"<>\" | \"<=\" | \">=\" | \"=\" | \"<\" | \">\" | \"!=\" )"`
	Value    *Value `parser:"@@"`
}

// Value represents a value that can be compared in a condition
type Value struct {
	Float   *float64  `parser:"( @Float "`
	String  *string   `parser:" | @String"`
	Regex   *RegexVal `parser:" | @Regex"`
	Boolean *bool     `parser:" | @(\"TRUE\" | \"FALSE\")"`
	Null    bool      `parser:" | @\"NULL\" )"`
}

// セキュリティのための定数
const (
	// MaxRegexPatternLength は正規表現パターンの最大長
	MaxRegexPatternLength = 1000
	// MaxRegexComplexity は正規表現の複雑さの最大値（繰り返し演算子の数）
	MaxRegexComplexity = 20
)

// RegexVal represents a regular expression pattern
type RegexVal struct {
	Pattern string
	Regexp  *regexp.Regexp
}

// Capture implements the participle.Capture interface for RegexVal
func (r *RegexVal) Capture(values []string) error {
	if len(values) == 0 {
		return errors.New("no regex pattern to capture")
	}
	
	// Remove the leading '/' and trailing '/' from the regex pattern
	pattern := values[0]
	if len(pattern) < 3 { // Need at least /x/
		return fmt.Errorf("invalid regex pattern: %s", pattern)
	}
	
	// Extract the pattern between slashes
	pattern = pattern[1 : len(pattern)-1]
	
	// エスケープされたスラッシュを処理
	// Go の文字列リテラル内では \\ は \ に変換され、\\/は \/ になる
	// 正規表現内では \/ はエスケープされたスラッシュを意味する
	pattern = strings.ReplaceAll(pattern, "\\/", "/")
	
	// セキュリティチェック: パターンの長さ制限
	if len(pattern) > MaxRegexPatternLength {
		return fmt.Errorf("regex pattern too long: %d characters (max %d)", len(pattern), MaxRegexPatternLength)
	}
	
	// セキュリティチェック: 複雑さの制限（繰り返し演算子の数をカウント）
	complexity := strings.Count(pattern, "*") + strings.Count(pattern, "+") + 
		strings.Count(pattern, "{") + strings.Count(pattern, "?") + 
		strings.Count(pattern, "|")
	if complexity > MaxRegexComplexity {
		return fmt.Errorf("regex pattern too complex: %d complexity score (max %d)", complexity, MaxRegexComplexity)
	}
	
	r.Pattern = pattern
	
	// Compile the regex pattern with timeout protection via context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// 非同期でコンパイルを実行
	ch := make(chan struct {
		re  *regexp.Regexp
		err error
	})
	go func() {
		// パターンをコンパイルする前にデバッグ出力
		fmt.Printf("Compiling regex pattern: %q\n", pattern)
		re, err := regexp.Compile(pattern)
		ch <- struct {
			re  *regexp.Regexp
			err error
		}{re, err}
	}()
	
	// タイムアウトまたは完了を待つ
	select {
	case result := <-ch:
		if result.err != nil {
			return fmt.Errorf("invalid regex pattern: %w", result.err)
		}
		r.Regexp = result.re
	case <-ctx.Done():
		return fmt.Errorf("regex compilation timed out: pattern may cause catastrophic backtracking")
	}
	
	return nil
}

// NewParser creates a new participle parser for parsing query expressions
func NewParser() *participle.Parser[Expression] {
	qLexer := lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Keyword", Pattern: `(?i)TRUE|FALSE|AND|OR|NULL`},
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
		{Name: "Float", Pattern: `[-+]?\d*\.?\d+([eE][-+]?\d+)?`},
		{Name: "String", Pattern: `'[^']*'|"[^"]*"`},
		{Name: "Regex", Pattern: `/[^/\\]*(\\.[^/\\]*)*/`}, // Regex pattern between slashes, allowing escaped characters
		{Name: "Operators", Pattern: `<>|!=|<=|>=|[-+*/%,.()=<>]`},
		{Name: "whitespace", Pattern: `\s+`},
	})
	return participle.MustBuild[Expression](
		participle.Lexer(qLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Keyword"),
		participle.UseLookahead(20),
	)
}

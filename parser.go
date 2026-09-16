package matcher

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Context is a map of string keys to arbitrary values that can be evaluated against expressions
type Context map[string]any

// ErrInvalidContext is returned when an invalid context is provided
var ErrInvalidContext = errors.New("invalid context")

// ErrInvalidValue is returned when a value cannot be properly compared
var ErrInvalidValue = errors.New("invalid value")

// ErrInvalidOperator is returned when an unknown operator is encountered
var ErrInvalidOperator = errors.New("invalid operator")

// Expression represents a parsed query expression with OR conditions
type Expression struct {
	Or []*OrCondition `parser:"@@ ( \"OR\" @@ )*"`
}

// Eval evaluates the expression against the provided context
// Returns true if any of the OR conditions evaluate to true
func (e *Expression) Eval(c Context) (bool, error) {
	return e.EvalContext(context.Background(), c)
}

// EvalContext evaluates the expression against the provided context,
// observing cancellation via the provided context.Context.
// Returns true if any of the OR conditions evaluate to true
func (e *Expression) EvalContext(ctx context.Context, c Context) (bool, error) {
	if e == nil || len(e.Or) == 0 {
		return false, nil
	}

	for _, x := range e.Or {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		result, err := x.EvalContext(ctx, c)
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
func (e *OrCondition) Eval(c Context) (bool, error) {
	return e.EvalContext(context.Background(), c)
}

// EvalContext evaluates the AND conditions against the provided context,
// observing cancellation via the provided context.Context.
// Returns true only if all AND conditions evaluate to true
func (e *OrCondition) EvalContext(ctx context.Context, c Context) (bool, error) {
	if e == nil || len(e.And) == 0 {
		return false, nil
	}

	for _, x := range e.And {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		result, err := x.EvalContext(ctx, c)
		if err != nil {
			return false, fmt.Errorf("evaluating AND condition: %w", err)
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

// Condition represents either a simple condition or a nested expression in
// parentheses, optionally negated with the NOT keyword
type Condition struct {
	Not bool `parser:"(@\"NOT\")? ("`
	// Only one of these will be set
	Nested    *Expression `parser:"  \"(\" @@ \")\""`
	Predicate *Predicate  `parser:"| @@ )"`
}

// Eval evaluates the condition against the provided context
func (x *Condition) Eval(c Context) (bool, error) {
	return x.EvalContext(context.Background(), c)
}

// EvalContext evaluates the condition against the provided context,
// observing cancellation via the provided context.Context.
func (x *Condition) EvalContext(ctx context.Context, c Context) (bool, error) {
	if x == nil {
		return false, errors.New("invalid condition")
	}

	var (
		result bool
		err    error
	)

	// If this is a nested expression in parentheses, evaluate it,
	// otherwise evaluate the predicate
	switch {
	case x.Nested != nil:
		result, err = x.Nested.EvalContext(ctx, c)
	case x.Predicate != nil:
		result, err = x.Predicate.EvalContext(ctx, c)
	default:
		return false, errors.New("invalid predicate")
	}
	if err != nil {
		return false, err
	}

	// NOT negates the condition result
	if x.Not {
		return !result, nil
	}
	return result, nil
}

// Predicate represents a simple condition with a symbol and comparison
type Predicate struct {
	Symbol  string   `parser:"@Ident"`
	Compare *Compare `parser:"@@"`
}

// Eval evaluates the predicate against the provided context
func (p *Predicate) Eval(c Context) (bool, error) {
	return p.EvalContext(context.Background(), c)
}

// EvalContext evaluates the predicate against the provided context,
// observing cancellation via the provided context.Context.
func (p *Predicate) EvalContext(ctx context.Context, c Context) (bool, error) {
	if p == nil || p.Compare == nil {
		return false, errors.New("invalid predicate")
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	sym := p.Symbol
	ctxVal, ok := c[sym]
	if !ok {
		// Symbol not found in context, return false but not an error
		return false, nil
	}

	// IN list membership: true if ctxVal equals any listed value
	if p.Compare.In != nil {
		for _, v := range p.Compare.In {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			ok, err := compareEqual(ctxVal, v)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}

	// CONTAINS: substring check for strings, element membership for slices
	if p.Compare.Contains != nil {
		return evalContains(ctx, ctxVal, p.Compare.Contains)
	}

	switch o := p.Compare.Operator; o {
	case "=":
		return compareEqual(ctxVal, p.Compare.Value)
	case "<>", "!=":
		v := p.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				f, _ := toFloat(x)
				return f != *v.Float, nil
			case string:
				f, err := strconv.ParseFloat(x, 64)
				if err != nil {
					return true, nil
				}
				return f != *v.Float, nil
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
					return false, fmt.Errorf("is not bool value: %s, %w", x, err)
				}
				return b != *v.Boolean, nil
			}
		case v.Null:
			return ctxVal != nil, nil
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case ">":
		v := p.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				f, _ := toFloat(x)
				return f > *v.Float, nil
			case string:
				f, err := strconv.ParseFloat(x, 64)
				if err != nil {
					return false, fmt.Errorf("cannot compare non-numeric string %q with number", x)
				}
				return f > *v.Float, nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
			}
		case v.String != nil:
			strVal, ok := ctxVal.(string)
			if !ok {
				return false, fmt.Errorf("cannot compare string with non-string value: %T", ctxVal)
			}
			return strVal > *v.String, nil
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use > operator with regex pattern")
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case ">=":
		v := p.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				f, _ := toFloat(x)
				return f >= *v.Float, nil
			case string:
				f, err := strconv.ParseFloat(x, 64)
				if err != nil {
					return false, fmt.Errorf("cannot compare non-numeric string %q with number", x)
				}
				return f >= *v.Float, nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
			}
		case v.String != nil:
			strVal, ok := ctxVal.(string)
			if !ok {
				return false, fmt.Errorf("cannot compare string with non-string value: %T", ctxVal)
			}
			return strVal >= *v.String, nil
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use >= operator with regex pattern")
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case "<":
		v := p.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				f, _ := toFloat(x)
				return f < *v.Float, nil
			case string:
				f, err := strconv.ParseFloat(x, 64)
				if err != nil {
					return false, fmt.Errorf("cannot compare non-numeric string %q with number", x)
				}
				return f < *v.Float, nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
			}
		case v.String != nil:
			strVal, ok := ctxVal.(string)
			if !ok {
				return false, fmt.Errorf("cannot compare string with non-string value: %T", ctxVal)
			}
			return strVal < *v.String, nil
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use < operator with regex pattern")
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	case "<=":
		v := p.Compare.Value
		switch {
		case v.Float != nil:
			switch x := ctxVal.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				f, _ := toFloat(x)
				return f <= *v.Float, nil
			case string:
				f, err := strconv.ParseFloat(x, 64)
				if err != nil {
					return false, fmt.Errorf("cannot compare non-numeric string %q with number", x)
				}
				return f <= *v.Float, nil
			case bool:
				return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
			}
		case v.String != nil:
			strVal, ok := ctxVal.(string)
			if !ok {
				return false, fmt.Errorf("cannot compare string with non-string value: %T", ctxVal)
			}
			return strVal <= *v.String, nil
		case v.Regex != nil:
			return false, fmt.Errorf("cannot use <= operator with regex pattern")
		case v.Boolean != nil:
			return false, fmt.Errorf("boolean did not compare by greater/less than: %#v", v)
		default:
			return false, fmt.Errorf("unknown value type: %#v", v)
		}

	default:
		return false, fmt.Errorf("unknown operator: %s", o)
	}
	return false, fmt.Errorf("failed to complete comparison, type: %T: %#v", ctxVal, ctxVal)
}

// compareEqual implements the equality semantics of the `=` operator.
// It is shared by `=` and by the IN and CONTAINS operators.
func compareEqual(ctxVal any, v *Value) (bool, error) {
	switch {
	case v.Float != nil:
		switch x := ctxVal.(type) {
		case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			f, _ := toFloat(x)
			return f == *v.Float, nil
		case string:
			f, err := strconv.ParseFloat(x, 64)
			if err != nil {
				return false, nil
			}
			return f == *v.Float, nil
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
				return false, fmt.Errorf("is not bool value: %s, %w", x, err)
			}
			return b == *v.Boolean, nil
		}
	case v.Null:
		return ctxVal == nil, nil
	default:
		return false, fmt.Errorf("unknown value type: %#v", v)
	}
	return false, fmt.Errorf("failed to complete comparison, type: %T: %#v", ctxVal, ctxVal)
}

// evalContains implements the CONTAINS operator: substring matching for string
// context values, element membership for slices/arrays (using the same
// equality semantics as IN). Other context value types are an error.
func evalContains(ctx context.Context, ctxVal any, v *Value) (bool, error) {
	if s, ok := ctxVal.(string); ok {
		if v.String == nil {
			return false, fmt.Errorf("CONTAINS on a string requires a string operand")
		}
		return strings.Contains(s, *v.String), nil
	}

	rv := reflect.ValueOf(ctxVal)
	if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
		for i := 0; i < rv.Len(); i++ {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			ok, err := compareEqual(rv.Index(i).Interface(), v)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}

	return false, fmt.Errorf("CONTAINS not supported for %T", ctxVal)
}

// toFloat converts a numeric value to float64 for comparison.
// Returns false if the value is not a numeric type.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float32:
		return float64(x), true
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}

// Compare represents a comparison operation.
// Exactly one of the three forms is set:
//   - Operator+Value: a scalar comparison (=, !=, <>, >, >=, <, <=)
//   - In: an IN list membership test
//   - Contains: a CONTAINS substring/element-membership test
type Compare struct {
	Operator string   `parser:"( @( \"<>\" | \"<=\" | \">=\" | \"=\" | \"<\" | \">\" | \"!=\" )"`
	Value    *Value   `parser:"    @@"`
	In       []*Value `parser:"  | \"IN\" \"(\" @@ ( \",\" @@ )* \")\""`
	Contains *Value   `parser:"  | \"CONTAINS\" @@ )"`
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
	
	token := values[0]
	if len(token) < 3 { // Need at least /x/
		return fmt.Errorf("invalid regex pattern: %s", token)
	}

	// Split off optional flags after the closing slash. The closing slash is
	// always the last '/' in the token since slashes inside the pattern must
	// be escaped (\/).
	idx := strings.LastIndex(token, "/")
	if idx <= 0 {
		return fmt.Errorf("invalid regex pattern: %s", token)
	}
	pattern := token[1:idx]
	flags := token[idx+1:]

	// Only the 'i' (case-insensitive) flag is supported.
	caseInsensitive := false
	for _, f := range flags {
		if f != 'i' {
			return fmt.Errorf("unsupported regex flag %q in %s (only 'i' is supported)", f, token)
		}
		caseInsensitive = true
	}

	// エスケープされたスラッシュを処理
	// Go の文字列リテラル内では \\ は \ に変換され、\\/は \/ になる
	// 正規表現内では \/ はエスケープされたスラッシュを意味する
	pattern = strings.ReplaceAll(pattern, "\\/", "/")

	// セキュリティチェック: パターンの長さ制限
	if len(pattern) > MaxRegexPatternLength {
		return fmt.Errorf("regex pattern too long: %d characters (max %d)", len(pattern), MaxRegexPatternLength)
	}

	r.Pattern = pattern

	if caseInsensitive {
		pattern = "(?i)" + pattern
	}

	// Go の regexp は RE2 ベースで後方参照や壊滅的バックトラッキングが
	// 発生しないため、タイムアウト付きの非同期コンパイルは不要
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	r.Regexp = re

	return nil
}

// NewParser creates a new participle parser for parsing query expressions
func NewParser() *participle.Parser[Expression] {
	qLexer := lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Keyword", Pattern: `(?i)(TRUE|FALSE|AND|OR|NOT|IN|CONTAINS|NULL)\b`},
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
		{Name: "Float", Pattern: `[-+]?\d*\.?\d+([eE][-+]?\d+)?`},
		{Name: "String", Pattern: `'[^']*'|"[^"]*"`},
		{Name: "Regex", Pattern: `/[^/\\]*(\\.[^/\\]*)*/[a-zA-Z]*`}, // Regex pattern between slashes, allowing escaped characters and trailing flags
		{Name: "Operators", Pattern: `<>|!=|<=|>=|[(),=<>]`},
		{Name: "whitespace", Pattern: `\s+`},
	})
	return participle.MustBuild[Expression](
		participle.Lexer(qLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Keyword"),
		participle.UseLookahead(20),
	)
}

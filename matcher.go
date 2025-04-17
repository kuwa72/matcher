package matcher

import (
	"context"
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/repr"
)

// Matcher represents a query matcher that evaluates expressions against a context.
type Matcher struct {
	Parser     *participle.Parser[Expression]
	Expression Expression
	Debug      bool
}

// NewMatcher creates a new matcher with the given query string.
func NewMatcher(q string) (*Matcher, error) {
	if q == "" {
		return nil, fmt.Errorf("empty query string")
	}

	parser := NewParser()
	expression, err := parser.ParseString("", q)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &Matcher{
		Parser:     parser,
		Expression: *expression, // Dereference the pointer to get the actual Expression value
		Debug:      false,
	}, nil
}

// Test evaluates the matcher's expression against the provided context.
func (m Matcher) Test(c *Context) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("nil context provided")
	}

	if m.Debug {
		repr.Println(m.Expression, repr.Indent("  "), repr.OmitEmpty(true))
	}

	return m.Expression.Eval(*c)
}

// TestWithContext evaluates the matcher's expression with a cancellable context.
func (m Matcher) TestWithContext(ctx context.Context, c *Context) (bool, error) {
	if ctx == nil {
		return false, fmt.Errorf("nil context.Context provided")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		// Continue with evaluation
	}

	return m.Test(c)
}

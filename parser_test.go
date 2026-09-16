package matcher_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/kuwa72/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleMatcher(t *testing.T) {
	cases := []struct {
		query string
		json  string
		match bool
	}{
		// =
		{"a=1", "{\"a\":1}", true},
		{"a=2", "{\"a\":1}", false},

		// <>, !=
		{"a<>2", "{\"a\":1}", true},
		{"a!=2", "{\"a\":2}", false},

		// >
		{"a>2", "{\"a\":3}", true},
		{"a>2", "{\"a\":2}", false},

		// >=
		{"a>=2", "{\"a\":3}", true},
		{"a>=2", "{\"a\":2}", true},
		{"a>=2", "{\"a\":1}", false},

		// <
		{"a<2", "{\"a\":1}", true},
		{"a<2", "{\"a\":2}", false},

		// <=
		{"a<=2", "{\"a\":3}", false},
		{"a<=2", "{\"a\":2}", true},
		{"a<=2", "{\"a\":1}", true},
	}

	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			assert := assert.New(t)
			m, err := matcher.NewMatcher(c.query)
			assert.NoError(err)

			ctx := make(matcher.Context)
			err = json.Unmarshal([]byte(c.json), &ctx)
			assert.NoError(err)

			ok, err := m.Test(&ctx)
			assert.Equal(c.match, ok)
			assert.NoError(err)
		})
	}

}

func TestComplexMatcher(t *testing.T) {
	cases := []struct {
		name  string
		query string
		json  string
		match bool
	}{
		{
			name:  "Complex AND/OR condition",
			query: "a=1 and a>0 and a >= 1 and b > 5 or c = \"foo\"", 
			json:  "{\"a\":1, \"b\":5.5, \"c\":\"foo\"}", 
			match: true,
		},
		{
			name:  "OR with comparison operators",
			query: "a <= 5 or b != 2", 
			json:  "{\"a\": 5, \"b\": 2, \"c\":1024}", 
			match: true,
		},
		{
			name:  "Missing field in context",
			query: "missing_field = 1", 
			json:  "{\"a\": 5, \"b\": 2}", 
			match: false,
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			
			m, err := matcher.NewMatcher(tc.query)
			require.NoError(err, "Failed to create matcher")

			ctxMap := make(matcher.Context)
			err = json.Unmarshal([]byte(tc.json), &ctxMap)
			require.NoError(err, "Failed to unmarshal JSON")

			// Test both regular and context-aware methods
			ok, err := m.Test(&ctxMap)
			assert.Equal(tc.match, ok)
			assert.NoError(err)

			// Test with context
			okWithCtx, err := m.TestWithContext(ctx, &ctxMap)
			assert.Equal(tc.match, okWithCtx)
			assert.NoError(err)
		})
	}
}

func TestParenthesesGrouping(t *testing.T) {
	cases := []struct {
		name  string
		query string
		json  string
		match bool
	}{
		{
			name:  "Simple parentheses",
			query: "(a = 1)", 
			json:  "{\"a\":1}", 
			match: true,
		},
		{
			name:  "Parentheses with AND",
			query: "(a = 1 AND b = 2)", 
			json:  "{\"a\":1, \"b\":2}", 
			match: true,
		},
		{
			name:  "Parentheses with OR",
			query: "(a = 1 OR b = 2)", 
			json:  "{\"a\":1, \"b\":3}", 
			match: true,
		},
		{
			name:  "Parentheses changing precedence",
			query: "a = 1 AND (b = 2 OR c = 3)", 
			json:  "{\"a\":1, \"b\":5, \"c\":3}", 
			match: true,
		},
		{
			name:  "Parentheses changing precedence - false case",
			query: "a = 1 AND (b = 2 OR c = 3)", 
			json:  "{\"a\":1, \"b\":5, \"c\":5}", 
			match: false,
		},
		{
			name:  "Multiple nested parentheses",
			query: "(a = 1 AND (b > 5 OR (c = 3 AND d = 4)))", 
			json:  "{\"a\":1, \"b\":3, \"c\":3, \"d\":4}", 
			match: true,
		},
		{
			name:  "Complex expression with parentheses",
			query: "(a = 1 OR a = 2) AND (b = 3 OR b = 4)", 
			json:  "{\"a\":2, \"b\":3}", 
			match: true,
		},
		{
			name:  "Precedence without vs with parentheses",
			query: "a = 1 OR b = 2 AND c = 3", // Equivalent to: a = 1 OR (b = 2 AND c = 3)
			json:  "{\"a\":0, \"b\":2, \"c\":3}", 
			match: true,
		},
		{
			name:  "Explicit precedence with parentheses",
			query: "(a = 1 OR b = 2) AND c = 3", 
			json:  "{\"a\":0, \"b\":2, \"c\":3}", 
			match: true,
		},
		{
			name:  "Explicit precedence with parentheses - false case",
			query: "(a = 1 OR b = 2) AND c = 3", 
			json:  "{\"a\":0, \"b\":2, \"c\":4}", 
			match: false,
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			
			m, err := matcher.NewMatcher(tc.query)
			require.NoError(err, "Failed to create matcher")

			ctxMap := make(matcher.Context)
			err = json.Unmarshal([]byte(tc.json), &ctxMap)
			require.NoError(err, "Failed to unmarshal JSON")

			// Test both regular and context-aware methods
			ok, err := m.Test(&ctxMap)
			assert.Equal(tc.match, ok)
			assert.NoError(err)

			// Test with context
			okWithCtx, err := m.TestWithContext(ctx, &ctxMap)
			assert.Equal(tc.match, okWithCtx)
			assert.NoError(err)
		})
	}
}

func TestRegexMatching(t *testing.T) {
	cases := []struct {
		name  string
		query string
		json  string
		match bool
	}{
		{
			name:  "Simple regex match",
			query: "name = /Tan.*/", 
			json:  "{\"name\":\"Tanya\"}", 
			match: true,
		},
		{
			name:  "Simple regex non-match",
			query: "name = /Tan.*/", 
			json:  "{\"name\":\"John\"}", 
			match: false,
		},
		{
			name:  "Regex with negation",
			query: "name != /Tan.*/", 
			json:  "{\"name\":\"John\"}", 
			match: true,
		},
		{
			name:  "Regex with negation - false case",
			query: "name != /Tan.*/", 
			json:  "{\"name\":\"Tanya\"}", 
			match: false,
		},
		{
			name:  "Regex with special characters",
			query: "email = /.*@.*\\.com$/", 
			json:  "{\"email\":\"user@example.com\"}", 
			match: true,
		},
		{
			name:  "Regex with special characters - false case",
			query: "email = /.*@.*\\.com$/", 
			json:  "{\"email\":\"user@example.org\"}", 
			match: false,
		},
		{
			name:  "Regex with character classes",
			query: "code = /[a-z][0-9]{3}/", 
			json:  "{\"code\":\"a123\"}", 
			match: true,
		},
		{
			name:  "Regex with character classes - false case",
			query: "code = /[a-z][0-9]{3}/", 
			json:  "{\"code\":\"A123\"}", 
			match: false,
		},
		{
			name:  "Regex with AND condition",
			query: "name = /J.*/ AND age > 30", 
			json:  "{\"name\":\"John\", \"age\": 35}", 
			match: true,
		},
		{
			name:  "Regex with OR condition",
			query: "name = /J.*/ OR age > 30", 
			json:  "{\"name\":\"Tanya\", \"age\": 35}", 
			match: true,
		},
		{
			name:  "Regex with parentheses grouping",
			query: "(name = /J.*/ OR name = /T.*/) AND age > 30", 
			json:  "{\"name\":\"Tanya\", \"age\": 35}", 
			match: true,
		},
		{
			name:  "Regex with non-string value",
			query: "age = /[0-9]+/", 
			json:  "{\"age\": 35}", 
			match: false, // Should fail because age is a number, not a string
		},
		{
			name:  "Regex with simple forward slash",
			query: "path = /foo\\/bar/", 
			json:  "{\"path\":\"foo/bar\"}", 
			match: true,
		},
		{
			name:  "Regex with simple forward slash - false case",
			query: "path = /foo\\/bar/", 
			json:  "{\"path\":\"foo\\\\bar\"}", 
			match: false,
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			
			m, err := matcher.NewMatcher(tc.query)
			require.NoError(err, "Failed to create matcher")

			ctxMap := make(matcher.Context)
			err = json.Unmarshal([]byte(tc.json), &ctxMap)
			require.NoError(err, "Failed to unmarshal JSON")

			// Test both regular and context-aware methods
			ok, err := m.Test(&ctxMap)
			if tc.name == "Regex with non-string value" {
				assert.Error(err, "Expected error for regex on non-string value")
			} else {
				assert.NoError(err)
				assert.Equal(tc.match, ok)
			}

			// Test with context
			okWithCtx, err := m.TestWithContext(ctx, &ctxMap)
			if tc.name == "Regex with non-string value" {
				assert.Error(err, "Expected error for regex on non-string value")
			} else {
				assert.NoError(err)
				assert.Equal(tc.match, okWithCtx)
			}
		})
	}
}

// TestNumericContextTypes is a regression test for issue #2: comparing a
// numeric literal against non-float64 numeric context values must not panic.
// JSON unmarshalling always produces float64, so contexts are built directly.
func TestNumericContextTypes(t *testing.T) {
	cases := []struct {
		name  string
		query string
		ctx   matcher.Context
		match bool
	}{
		// = with various numeric types
		{name: "int equality", query: "a = 1", ctx: matcher.Context{"a": int(1)}, match: true},
		{name: "int64 equality", query: "a = 1", ctx: matcher.Context{"a": int64(1)}, match: true},
		{name: "float32 equality", query: "a = 1", ctx: matcher.Context{"a": float32(1)}, match: true},
		{name: "uint8 equality", query: "a = 1", ctx: matcher.Context{"a": uint8(1)}, match: true},
		{name: "int64 inequality", query: "a = 2", ctx: matcher.Context{"a": int64(1)}, match: false},

		// != with various numeric types
		{name: "int64 not-equal true", query: "a != 2", ctx: matcher.Context{"a": int64(1)}, match: true},
		{name: "uint not-equal false", query: "a != 1", ctx: matcher.Context{"a": uint(1)}, match: false},

		// ordering operators with various numeric types
		{name: "int greater than", query: "a > 0", ctx: matcher.Context{"a": int(1)}, match: true},
		{name: "int64 greater than", query: "a > 0", ctx: matcher.Context{"a": int64(1)}, match: true},
		{name: "uint greater than", query: "a > 0", ctx: matcher.Context{"a": uint(1)}, match: true},
		{name: "int64 greater than or equal", query: "a >= 1", ctx: matcher.Context{"a": int64(1)}, match: true},
		{name: "int32 less than", query: "a < 2", ctx: matcher.Context{"a": int32(1)}, match: true},
		{name: "uint16 less than or equal", query: "a <= 1", ctx: matcher.Context{"a": uint16(1)}, match: true},
		{name: "int8 less than false", query: "a < 1", ctx: matcher.Context{"a": int8(1)}, match: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			m, err := matcher.NewMatcher(tc.query)
			require.NoError(err, "Failed to create matcher")

			ok, err := m.Test(&tc.ctx)
			assert.NoError(err)
			assert.Equal(tc.match, ok)
		})
	}
}

// TestStringOrderingWithNonString is a regression test for issue #3: ordering
// comparisons between a string literal and a non-string context value must
// return an error instead of panicking.
func TestStringOrderingWithNonString(t *testing.T) {
	cases := []struct {
		name  string
		query string
		ctx   matcher.Context
	}{
		{name: "greater than", query: `a > "x"`, ctx: matcher.Context{"a": 123}},
		{name: "greater than or equal", query: `a >= "x"`, ctx: matcher.Context{"a": 123}},
		{name: "less than", query: `a < "x"`, ctx: matcher.Context{"a": 123}},
		{name: "less than or equal", query: `a <= "x"`, ctx: matcher.Context{"a": 123}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			m, err := matcher.NewMatcher(tc.query)
			require.NoError(err, "Failed to create matcher")

			_, err = m.Test(&tc.ctx)
			assert.Error(err, "Expected error comparing string with non-string value")
		})
	}
}

// TestFloatLiteralVsStringContext is a regression test for issue #7: a numeric
// literal compared against a string context value must parse the string and
// compare numerically instead of comparing against a "%f"-formatted string.
func TestFloatLiteralVsStringContext(t *testing.T) {
	cases := []struct {
		name    string
		query   string
		ctx     matcher.Context
		match   bool
		wantErr bool
	}{
		{name: "equal numeric string", query: "a = 1", ctx: matcher.Context{"a": "1"}, match: true},
		{name: "equal non-numeric string", query: "a = 1", ctx: matcher.Context{"a": "x"}, match: false},
		{name: "not-equal non-numeric string", query: "a != 1", ctx: matcher.Context{"a": "x"}, match: true},
		{name: "not-equal numeric string", query: "a != 1", ctx: matcher.Context{"a": "1"}, match: false},
		{name: "not-equal numeric string <>", query: "a <> 1", ctx: matcher.Context{"a": "2"}, match: true},
		{name: "greater than numeric string", query: "a > 0", ctx: matcher.Context{"a": "2"}, match: true},
		{name: "greater than numeric string false", query: "a > 5", ctx: matcher.Context{"a": "2"}, match: false},
		{name: "greater than non-numeric string", query: "a > 1", ctx: matcher.Context{"a": "x"}, wantErr: true},
		{name: "greater-or-equal non-numeric string", query: "a >= 1", ctx: matcher.Context{"a": "x"}, wantErr: true},
		{name: "less than numeric string", query: "a < 10", ctx: matcher.Context{"a": "2"}, match: true},
		{name: "less than non-numeric string", query: "a < 1", ctx: matcher.Context{"a": "x"}, wantErr: true},
		{name: "less-or-equal numeric string", query: "a <= 2", ctx: matcher.Context{"a": "2"}, match: true},
		{name: "less-or-equal non-numeric string", query: "a <= 1", ctx: matcher.Context{"a": "x"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			m, err := matcher.NewMatcher(tc.query)
			require.NoError(err, "Failed to create matcher")

			ok, err := m.Test(&tc.ctx)
			if tc.wantErr {
				assert.Error(err, "Expected error comparing non-numeric string with number")
			} else {
				assert.NoError(err)
				assert.Equal(tc.match, ok)
			}
		})
	}
}

// TestWithContextCancellation is a regression test for issue #9:
// TestWithContext must return the context error when the context is cancelled.
func TestWithContextCancellation(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	m, err := matcher.NewMatcher("a = 1")
	require.NoError(err, "Failed to create matcher")

	c := matcher.Context{"a": 1}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok, err := m.TestWithContext(ctx, &c)
	assert.ErrorIs(err, context.Canceled)
	assert.False(ok)
}

// TestNotInContains covers issue #15: the NOT, IN and CONTAINS operators.
func TestNotInContains(t *testing.T) {
	cases := []struct {
		name    string
		query   string
		ctx     matcher.Context
		match   bool
		wantErr bool
	}{
		// NOT
		{name: "NOT true case", query: "NOT a = 1", ctx: matcher.Context{"a": 2}, match: true},
		{name: "NOT false case", query: "NOT a = 1", ctx: matcher.Context{"a": 1}, match: false},
		{name: "NOT grouped OR", query: "NOT (a = 1 OR b = 2)", ctx: matcher.Context{"a": 0, "b": 0}, match: true},
		{name: "NOT grouped OR false", query: "NOT (a = 1 OR b = 2)", ctx: matcher.Context{"a": 0, "b": 2}, match: false},
		{name: "lowercase not", query: "not a = 1", ctx: matcher.Context{"a": 1}, match: false},
		{name: "NOT with AND", query: "NOT a = 1 AND b = 2", ctx: matcher.Context{"a": 2, "b": 2}, match: true},
		{name: "NOT IN list", query: "NOT a IN (1, 2)", ctx: matcher.Context{"a": 3}, match: true},

		// IN
		{name: "IN numbers match", query: "a IN (1, 2, 3)", ctx: matcher.Context{"a": 2}, match: true},
		{name: "IN numbers no match", query: "a IN (1, 2, 3)", ctx: matcher.Context{"a": 5}, match: false},
		{name: "IN strings match", query: `s IN ("x", "y")`, ctx: matcher.Context{"s": "y"}, match: true},
		{name: "IN strings no match", query: `s IN ("x", "y")`, ctx: matcher.Context{"s": "z"}, match: false},
		{name: "IN with int64 context value", query: "a IN (1, 2)", ctx: matcher.Context{"a": int64(2)}, match: true},
		{name: "lowercase in", query: "a in (1, 2)", ctx: matcher.Context{"a": 1}, match: true},
		{name: "IN missing field", query: "missing IN (1, 2)", ctx: matcher.Context{"a": 1}, match: false},

		// CONTAINS
		{name: "CONTAINS substring", query: `tags CONTAINS "o"`, ctx: matcher.Context{"tags": "gopher"}, match: true},
		{name: "CONTAINS substring no match", query: `tags CONTAINS "x"`, ctx: matcher.Context{"tags": "gopher"}, match: false},
		{name: "CONTAINS slice element", query: `arr CONTAINS "x"`, ctx: matcher.Context{"arr": []any{"x", "y"}}, match: true},
		{name: "CONTAINS slice no match", query: `arr CONTAINS "z"`, ctx: matcher.Context{"arr": []any{"x", "y"}}, match: false},
		{name: "CONTAINS typed slice", query: "arr CONTAINS 2", ctx: matcher.Context{"arr": []int{1, 2, 3}}, match: true},
		{name: "lowercase contains", query: `tags contains "o"`, ctx: matcher.Context{"tags": "gopher"}, match: true},
		{name: "CONTAINS on number errors", query: `n CONTAINS "x"`, ctx: matcher.Context{"n": 123}, wantErr: true},
		{name: "CONTAINS non-string operand on string errors", query: `tags CONTAINS 1`, ctx: matcher.Context{"tags": "gopher"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			m, err := matcher.NewMatcher(tc.query)
			require.NoError(err, "Failed to create matcher")

			ok, err := m.Test(&tc.ctx)
			if tc.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.match, ok)
			}
		})
	}
}

// TestRegexFlags covers issue #16: case-insensitive regex matching with the
// /pattern/i flag and rejection of unknown flags.
func TestRegexFlags(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		ctx      matcher.Context
		match    bool
		parseErr bool
	}{
		{name: "i flag matches different case", query: "name = /john/i", ctx: matcher.Context{"name": "John"}, match: true},
		{name: "i flag matches same case", query: "name = /john/i", ctx: matcher.Context{"name": "johnny"}, match: true},
		{name: "i flag non-match", query: "name = /john/i", ctx: matcher.Context{"name": "Tanya"}, match: false},
		{name: "no flag stays case-sensitive", query: "name = /john/", ctx: matcher.Context{"name": "John"}, match: false},
		{name: "i flag with negation", query: "name != /john/i", ctx: matcher.Context{"name": "John"}, match: false},
		{name: "i flag with pattern metacharacters", query: `name = /^(john|jane)$/i`, ctx: matcher.Context{"name": "JANE"}, match: true},
		{name: "invalid flag is a parse error", query: "name = /x/z", ctx: matcher.Context{"name": "x"}, parseErr: true},
		{name: "multi-letter invalid flag is a parse error", query: "name = /x/ix", ctx: matcher.Context{"name": "x"}, parseErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			m, err := matcher.NewMatcher(tc.query)
			if tc.parseErr {
				assert.Error(err, "Expected parse error")
				return
			}
			require.NoError(err, "Failed to create matcher")

			ok, err := m.Test(&tc.ctx)
			assert.NoError(err)
			assert.Equal(tc.match, ok)
		})
	}
}

func BenchmarkComplexMatcher(b *testing.B) {
	b.ReportAllocs()
	
	m, err := matcher.NewMatcher("index = 0 and balance = \"$1,713.88\" and age = 40 and latitude = -63.183265")
	require.NoError(b, err, "Failed to create matcher")

	ctx := make(matcher.Context)
	content, err := os.ReadFile("testfiles/example.json")
	require.NoError(b, err, "Failed to read test file")

	err = json.Unmarshal(content, &ctx)
	require.NoError(b, err, "Failed to unmarshal JSON")

	// Run with context for timeout support
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := m.TestWithContext(ctxWithTimeout, &ctx)
		require.NoError(b, err)
	}
}

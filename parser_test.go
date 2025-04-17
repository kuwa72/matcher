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

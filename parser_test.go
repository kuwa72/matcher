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

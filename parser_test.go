package matcher_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/kuwa72/matcher"
	"github.com/stretchr/testify/assert"
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
		query string
		json  string
		match bool
	}{
		{"a=1 and a>0 and a >= 1 and b > 5 or c = \"foo\"", "{\"a\":1, \"b\":5.5, \"c\":\"foo\"}", true},
		{"a <= 5 or b != 2", "{\"a\": 5, \"b\": 2, \"c\":1024}", true},
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

func BenchmarkComplexMatcher(b *testing.B) {
	m, _ := matcher.NewMatcher("index = 0 and balance = \"$1,713.88\" and age = 40 and latitude = -63.183265")

	ctx := make(matcher.Context)
	content, _ := ioutil.ReadFile("testfile/example.json")

	json.Unmarshal([]byte(content), &ctx)

	for i := 0; i < b.N; i++ {
		m.Test(&ctx)
	}
}

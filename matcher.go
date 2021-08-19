package matcher

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/repr"
)

type Matcher struct {
	Parser     *participle.Parser
	Expression *Expression
	Debug      bool
}

func NewMatcher(q string) (*Matcher, error) {
	e := &Expression{}
	parser := NewParser()
	err := parser.ParseString("", q, e)
	return &Matcher{Parser: parser,
		Expression: e,
		Debug:      false}, err
}

func (m Matcher) Test(c *Context) (bool, error) {
	if m.Debug {
		repr.Println(m.Expression, repr.Indent("  "), repr.OmitEmpty(true))
	}
	return m.Expression.Eval(*c)
}

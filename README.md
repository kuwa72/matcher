# matcher

Is simple query language for go struct data, included JSON query tool.

# usage

## library

Constructor and test method, only for use.

```
package main

import (
	"fmt"
	"os"

	"github.com/kuwa72/matcher"
)

func main() {
	query := "a=1"
	m, err := matcher.NewMatcher(query) // constructor
	if err != nil {
		os.Exit(1)
	}
	m.Debug = true

	ctx := matcher.Context{"a": 1}

	ret, err := m.Test(&ctx) // check match data and query
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("matched?: %v", ret)
}
```

## cli

Install

```
go install github.com/kuwa72/matcher/matcher-cli
```

If input JSON(from stdin) and query(command argument) matched, return 0, otherwise 1.

example.

```
$ echo '{"a":1,"b":2,"c":"hoge"}' | matcher-cli 'b = 2 and a = 1 and a >= -1 and c = "hoge"'
```

# query

Dead simple.

`Identify Condition Value (Operator Identify Condition Value...)` like `a = 1 and b = "foo"`

* Operators: `AND, OR`
* Conditions: `=, !=(<>), >, >=, <, <=`
* Supported value type: Numbers(convert to float), String

Examples see test file: http://github.com/kuwa72/matcher/parser_test.go.

# license

MIT License

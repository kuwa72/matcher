# Matcher

A simple, efficient query language for Go struct data with JSON support. Matcher allows you to evaluate expressions against structured data with a SQL-like syntax.

## Usage

### Library

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kuwa72/matcher"
)

func main() {
	// Create a matcher with a query expression
	query := "a=1 AND b>5 OR c='foo'"
	m, err := matcher.NewMatcher(query)
	if err != nil {
		log.Fatalf("Failed to create matcher: %v", err)
	}

	// Enable debug mode to see the parsed expression
	m.Debug = true

	// Create a context with data to match against
	ctx := matcher.Context{
		"a": 1,
		"b": 10,
		"c": "foo",
	}

	// Basic matching
	matched, err := m.Test(&ctx)
	if err != nil {
		log.Fatalf("Error during matching: %v", err)
	}
	fmt.Printf("Basic match result: %v\n", matched)

	// With timeout context
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Context-aware matching with timeout
	matched, err = m.TestWithContext(timeoutCtx, &ctx)
	if err != nil {
		log.Fatalf("Error during matching with context: %v", err)
	}
	fmt.Printf("Context-aware match result: %v\n", matched)
}
```

### CLI Tool

#### Installation

```bash
go install github.com/kuwa72/matcher/matcher-cli@latest
```

#### Usage

The CLI tool reads JSON data from stdin and evaluates it against the query provided as a command-line argument. It returns exit code 0 if matched, 1 otherwise.

```bash
# Basic usage
$ echo '{"a":1,"b":2,"c":"hoge"}' | matcher-cli 'b = 2 and a = 1 and c = "hoge"'

# With debug output
$ echo '{"a":1,"b":2,"c":"hoge"}' | matcher-cli --debug 'b = 2 and a = 1 and c = "hoge"'

# With custom timeout (in seconds)
$ echo '{"a":1,"b":2,"c":"hoge"}' | matcher-cli --timeout 10 'b = 2 and a = 1 and c = "hoge"'
```

## Query Language

The query language is simple and intuitive, resembling SQL WHERE clauses.

### Syntax

`Identifier Condition Value (Operator Identifier Condition Value...)` 

For example: `a = 1 AND b = "foo" OR c > 5`

### Features

* **Logical Operators**: `AND`, `OR` (case-insensitive)
* **Comparison Operators**: `=`, `!=`, `<>`, `>`, `>=`, `<`, `<=`
* **Grouping**: Parentheses `()` for controlling evaluation order
* **Value Types**:
  * **Numbers**: Integers and floating-point (automatically converted to float64)
  * **Strings**: Enclosed in single or double quotes
  * **Booleans**: `TRUE` or `FALSE` (case-insensitive)
  * **NULL**: Special value for null checks

### Operator Precedence

1. Comparisons (`=`, `!=`, etc.) are evaluated first
2. `AND` conditions are evaluated next
3. `OR` conditions are evaluated last

### Examples

```
# Simple equality
a = 1

# Multiple conditions with AND
a = 1 AND b > 5 AND c = "string value"

# Using OR for alternatives
a = 1 OR b = 2

# Using parentheses for grouping
(a = 1 OR b = 2) AND c = 3

# Nested parentheses
(a = 1 AND (b > 5 OR (c = 3 AND d = 4)))

# Changing precedence with parentheses
a = 1 OR b = 2 AND c = 3  # Equivalent to: a = 1 OR (b = 2 AND c = 3)
(a = 1 OR b = 2) AND c = 3  # Different precedence with parentheses
```

For more examples, see the [test files](https://github.com/kuwa72/matcher/blob/main/parser_test.go).

## Performance

The matcher is designed to be efficient for evaluating expressions against large data structures. Benchmarks are included in the test suite.

## Requirements

- Go 1.22 or higher

## License

MIT License

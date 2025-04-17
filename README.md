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
  * **Regular Expressions**: Patterns enclosed in forward slashes `/pattern/`
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

# Regular expression matching
name = /John.*/  # Matches any string starting with "John"
email = /.*@example\.com$/  # Matches strings ending with "@example.com"

# Regular expression with negation
name != /Temp.*/  # Matches any string NOT starting with "Temp"

# Combining regex with other conditions
(name = /John.*/ OR name = /Jane.*/) AND age > 30
```

For more examples, see the [test files](https://github.com/kuwa72/matcher/blob/main/parser_test.go).

## Regular Expression Support

The matcher supports regular expression pattern matching for string values. Regular expressions must be enclosed in forward slashes (`/pattern/`).

### Security Features

The matcher implements several security features to prevent potential denial-of-service attacks through malicious regex patterns:

1. **Pattern Length Limit**: Regex patterns are limited to 1000 characters
2. **Complexity Limit**: Patterns with more than 20 repetition operators (`*`, `+`, `{...}`, `?`, `|`) are rejected
3. **Compilation Timeout**: Regex compilation is limited to 100ms to prevent catastrophic backtracking
4. **Asynchronous Processing**: Regex compilation runs in a separate goroutine to avoid blocking the main thread

### Syntax

```
field = /pattern/   # Matches if the field value matches the regex pattern
field != /pattern/  # Matches if the field value does NOT match the regex pattern
```

### Notes

- Regular expressions use Go's standard `regexp` package syntax
- Regex patterns can only be used with the equality (`=`) and inequality (`!=`, `<>`) operators
- Regex patterns can only be applied to string values
- Regex patterns can be combined with other conditions using logical operators and parentheses
- To include forward slashes in regex patterns, escape them with a backslash (`\/`)

### Examples

```
# Match email address format
email = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

# Match specific pattern at start of string
name = /^John/

# Match specific pattern at end of string
filename = /\.jpg$/

# Match any occurrence of a pattern
description = /contains this phrase/

# Match URL paths with forward slashes
path = /\/api\/v1\/users/

# Match file paths
filepath = /\/home\/[a-z]+\/.*\.txt/

# Match URLs
url = /https:\/\/[^\/]+\/[^\/]+/
```

## Performance

The matcher is designed to be efficient for evaluating expressions against large data structures. Comprehensive benchmarks are included in the test suite.

### Benchmark Results

The following benchmarks were performed on a dataset of 10,000 records, each containing 20 fields. The tests were run on an AMD Ryzen 9 5900HS processor.

#### Complex Query Performance

```
BenchmarkComplexQueryWithLargeDataset-16    5    215798360 ns/op    47335430 B/op    1095634 allocs/op
```

This benchmark evaluates a complex query with the following characteristics:
- Regular expression patterns (e.g., `/^J.*/`, `/\/api\/v[0-9]\/.*/`)
- Parentheses for grouping expressions
- Multiple logical operators (AND, OR)
- Various comparison operators
- Query: `(name = /^J.*/ OR department = "Engineering") AND (age > 30 AND salary >= 70000) AND (status = "Active" OR status = "Pending") AND path = /\/api\/v[0-9]\/.*/ AND score > 50`

The query matched approximately 0.5% of the 10,000 records (about 50-60 records).

#### Filtering Performance

```
BenchmarkFilteringWithLargeDataset-16    1    1325055000 ns/op    282463792 B/op    6580071 allocs/op
```

This benchmark evaluates multiple filtering queries against the same dataset:
- Simple numeric comparisons
- Regular expression matching
- Boolean field checks
- Compound conditions with parentheses

### Performance Considerations

1. **Memory Usage**: The matcher allocates memory during evaluation, primarily for context handling and regex compilation.
2. **Regex Compilation**: Regular expressions are compiled once when the matcher is created, not on each evaluation.
3. **Security Limits**: The matcher includes built-in limits for regex pattern length and complexity to prevent DoS attacks.
4. **Timeout Handling**: All regex operations have a built-in timeout to prevent catastrophic backtracking.

### Optimization Tips

1. **Prefer Simple Patterns**: When possible, use simple comparison operators instead of regex patterns.
2. **Limit Regex Complexity**: Keep regex patterns as simple as possible to minimize compilation and matching time.
3. **Order Conditions Efficiently**: Place conditions that are likely to fail first in AND expressions to take advantage of short-circuit evaluation.
4. **Reuse Matchers**: Create matchers once and reuse them for multiple evaluations rather than recreating them for each test.

## Requirements

- Go 1.22 or higher

## License

MIT License

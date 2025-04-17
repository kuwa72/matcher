# Matcher

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> **Powerful, flexible, and secure query language for filtering Go data structures**

Matcher is a high-performance Go library that lets you filter data structures using a simple yet powerful query language. It supports complex expressions with regex patterns, logical operators, and parentheses grouping - perfect for filtering JSON data, in-memory collections, or implementing query capabilities in your APIs.

*[日本語版はこちら](README-ja.md)*

## ✨ Highlights

- **Intuitive Query Language** - SQL-like syntax that's easy to learn and use
- **Powerful Regex Support** - Match string patterns with full regex capabilities
- **Parentheses Grouping** - Build complex nested expressions with precise control
- **High Performance** - Optimized for speed with minimal allocations
- **Security Built-in** - Protection against ReDoS attacks and resource exhaustion
- **Context Support** - Cancel long-running operations with timeouts

## 🚀 Quick Start

### Installation

```bash
go get github.com/kuwa72/matcher
```

### Basic Example

```go
package main

import (
	"fmt"
	"github.com/kuwa72/matcher"
)

func main() {
	// Create a matcher with a query string
	m, err := matcher.NewMatcher(`name = "John" AND age > 30`)
	if err != nil {
		panic(err)
	}

	// Data to test against
	data := matcher.Context{
		"name": "John",
		"age":  35,
	}

	// Test if the data matches the query
	result, err := m.Test(&data)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Match: %v\n", result) // Output: Match: true
}
```

### CLI Tool

```bash
# Installation
go install github.com/kuwa72/matcher/matcher-cli@latest

# Basic usage
echo '{"name":"John","age":35}' | matcher-cli 'name = "John" AND age > 30'

# Debug output
echo '{"name":"John","age":35}' | matcher-cli --debug 'name = "John" AND age > 30'
```

## 🔍 Query Language

Matcher uses an intuitive query language that's easy to learn yet powerful enough for complex filtering needs.

### Key Features

* **Logical Operators**: `AND`, `OR` (case-insensitive)
* **Comparison Operators**: `=`, `!=`, `<>`, `>`, `>=`, `<`, `<=`
* **Grouping**: Parentheses `()` for precise control over evaluation order
* **Value Types**:
  * **Numbers**: Integers and floating-point values
  * **Strings**: Enclosed in single or double quotes
  * **Regular Expressions**: Patterns enclosed in `/pattern/`
  * **Booleans**: `TRUE` or `FALSE` (case-insensitive)
  * **NULL**: Special value for null checks

### Operator Precedence

1. Comparisons (`=`, `!=`, etc.) are evaluated first
2. `AND` conditions are evaluated next
3. `OR` conditions are evaluated last

### 📝 Example Queries

```
# Simple equality
age = 30

# Multiple conditions with AND
name = "John" AND age > 30 AND status = "active"

# Using OR for alternatives
country = "USA" OR country = "Canada"

# Parentheses for grouping
(status = "pending" OR status = "approved") AND created_at > "2025-01-01"

# Complex nested expressions
(category = "electronics" AND (price < 1000 OR rating > 4.5)) OR featured = TRUE

# Regular expression matching
email = /.*@gmail\.com$/    # Match Gmail addresses
name = /^(John|Jane).*/     # Names starting with John or Jane

# Regex with forward slashes
path = /\/api\/v1\/.*/      # Match API v1 paths
url = /https:\/\/.*/       # Match HTTPS URLs

# Combining everything
(name = /J.*/ OR department = "Engineering") AND 
(age > 30 AND salary >= 70000) AND 
(status = "Active" OR status = "Pending")
```

See the [test files](https://github.com/kuwa72/matcher/blob/main/parser_test.go) for more examples.

## 🔒 Regular Expression Support

Matcher provides powerful regex pattern matching for string values, with built-in security protections.

### 🛡️ Security Features

All regex operations include protection against ReDoS attacks and resource exhaustion:

- **Pattern Length Limit**: Maximum 1000 characters per pattern
- **Complexity Limit**: No more than 20 repetition operators (`*`, `+`, `{...}`, `?`, `|`)
- **Compilation Timeout**: 100ms timeout prevents catastrophic backtracking
- **Asynchronous Processing**: Non-blocking compilation in separate goroutines

### 📋 Syntax

```
field = /pattern/   # Match if field matches the pattern
field != /pattern/  # Match if field does NOT match the pattern
```

### 📌 Important Notes

- Uses Go's standard `regexp` package syntax
- Works with equality (`=`) and inequality (`!=`, `<>`) operators
- Applies only to string values
- Escape forward slashes with backslash (`\/`)

### 🌟 Regex Examples

```go
// Email validation
email = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

// URL path matching
path = /\/api\/v[0-9]\/users/

// File extensions
filename = /\.(jpg|png|gif)$/

// Phone numbers
phone = /^\+?[0-9]{10,15}$/

// Complex patterns with escaping
url = /https:\/\/[^\/]+\/[^\/]+/
```

## ⚡ Performance

Matcher is designed for high performance even with large datasets and complex queries.

### 📊 Benchmark Results

Tested on 10,000 records (each with 20 fields) on an AMD Ryzen 9 5900HS:

#### Complex Query Performance

```
BenchmarkComplexQueryWithLargeDataset-16    5    215ms/op    47MB/op    1,095,634 allocs/op
```

**Query tested:**
```
(name = /^J.*/ OR department = "Engineering") AND 
(age > 30 AND salary >= 70000) AND 
(status = "Active" OR status = "Pending") AND 
path = /\/api\/v[0-9]\/.*/ AND score > 50
```

#### Multiple Filters Performance

```
BenchmarkFilteringWithLargeDataset-16    1    1,325ms/op    282MB/op    6,580,071 allocs/op
```

### 🔧 Optimization Tips

1. **Reuse Matchers** - Create once, reuse many times
2. **Prefer Simple Comparisons** - Use `=`, `>`, `<` instead of regex when possible
3. **Optimize Query Order** - Put likely-to-fail conditions first in AND expressions
4. **Limit Regex Complexity** - Simpler patterns perform better

## 🔧 Advanced Usage

### Context Support

Matcher supports Go's context package for timeout and cancellation:

```go
// Create a context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Test with context
result, err := matcher.TestWithContext(ctx, &data)
```

### JSON Integration

Matcher works seamlessly with JSON data:

```go
// Parse JSON data
var data matcher.Context
json.Unmarshal([]byte(`{"name":"John","age":35}`), &data)

// Create matcher
matcher, _ := matcher.NewMatcher(`name = "John" AND age > 30`)

// Test against JSON data
result, _ := matcher.Test(&data)
```

## 📦 Requirements

- Go 1.22 or higher

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

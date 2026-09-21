---
name: matcher
description: "Write and evaluate matcher query-language expressions against JSON/Go data via the Go API or matcher-cli. Use when constructing, debugging, or fixing matcher queries."
---

# matcher

`github.com/kuwa72/matcher` is a Go library + CLI that evaluates an SQL-like query language against flat `map[string]any` data.

## Query Language

### Operators

| Kind | Syntax |
|---|---|
| Comparison | `=` `!=` `<>` `>` `>=` `<` `<=` |
| Membership | `field IN (v1, v2, ...)` |
| Containment | `field CONTAINS value` — substring match on strings, element match on slices/arrays |
| Logical | `AND` `OR` `NOT` (case-insensitive) |
| Grouping | `( expr )` |

### Value literals

| Type | Syntax | Notes |
|---|---|---|
| Number | `30`, `-1.5`, `2e3` | parsed as `float64` |
| String | `'abc'` or `"abc"` | no escape processing inside quotes |
| Regex | `/pattern/`, `/pattern/i` | `i` = case-insensitive; escape `/` as `\/`; RE2 syntax; max 1000 chars; only with `=` `!=` `<>` |
| Boolean | `TRUE` / `FALSE` | case-insensitive |
| Null | `NULL` | `field = NULL` is true when the key exists and is `nil` |

### Precedence (high → low)

1. `NOT` and comparisons
2. `AND`
3. `OR`

`a = 1 OR b = 2 AND c = 3` parses as `a = 1 OR (b = 2 AND c = 3)`. Use parentheses to override.

## Go API

```go
import "github.com/kuwa72/matcher"

m, err := matcher.NewMatcher(`name = "John" AND age > 30`) // parse once, reuse
data := matcher.Context{"name": "John", "age": 35}         // type Context map[string]any

ok, err := m.Test(&data)                        // Test(*Context) (bool, error)
ok, err := m.TestWithContext(ctx, &data)        // cancellable variant
m.Debug = true                                  // dumps parsed AST via repr.Println
```

- `NewMatcher("")` errors; parse errors are wrapped as `parse error: ...`.
- A symbol missing from the context evaluates to `false`, not an error.
- Symbols match `Ident` `[a-zA-Z_][a-zA-Z0-9_]*` — top-level keys only, no dotted/nested paths.

## CLI

```bash
echo '{"name":"John","age":35}' | matcher-cli 'name = "John" AND age > 30'
```

- Reads one JSON object from stdin; prints `matched`/`unmatched`.
- Exit codes (grep convention): `0` matched, `1` unmatched, `2` error (bad query, bad JSON, eval error, timeout).
- Flags: `--debug` (AST + echo query/JSON), `--timeout N` (seconds, default 30).

## Pitfalls

- **Reserved keywords**: `NOT` `IN` `CONTAINS` `AND` `OR` `TRUE` `FALSE` `NULL` (any case) cannot be field names — they lex as keywords.
- **Space after regex**: the lexer swallows trailing letters as flags, so `a=/re/or b=2` fails (`or` → unsupported flag). Write `a = /re/ OR b = 2`.
- **RE2 only**: Go `regexp` — no backreferences or lookarounds. Only the `i` flag is supported.
- **Numeric coercion**: all Go int/uint/float types are normalized via `toFloat`; string context values that parse as numbers compare numerically against float literals (`age = 30` matches `"age": "30"`).
- **Bool coercion**: `flag = TRUE` also matches `int` 0/1 and strings parseable by `strconv.ParseBool`.
- `CONTAINS` on a string field requires a string operand; on other non-slice types it errors.
- `>` `>=` `<` `<=` on bool/regex operands error; string literals compare lexicographically (`"10" < "2"`).

## Examples

| Query | JSON | Result |
|---|---|---|
| `age >= 30 AND status = "active"` | `{"age":35,"status":"active"}` | matched |
| `email = /.*@gmail\.com$/i` | `{"email":"a@GMAIL.com"}` | matched |
| `tags CONTAINS "go"` | `{"tags":["go","cli"]}` | matched |
| `role IN ("admin","ops")` | `{"role":"dev"}` | unmatched |
| `NOT (a = 1 OR b = 2)` | `{"a":1}` | unmatched |
| `deleted_at = NULL` | `{"deleted_at":null}` | matched |

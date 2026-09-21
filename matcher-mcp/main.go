// matcher-mcp is an MCP server (stdio transport) that exposes the matcher
// query language to AI agents. It provides two tools:
//
//   - match:          evaluate a query against a JSON object
//   - validate_query: check a query for syntax errors
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kuwa72/matcher"
)

// Safety limits. Defaults may be overridden via environment variables.
var (
	maxQueryLength = envInt("MATCHER_MCP_MAX_QUERY_LENGTH", 4*1024)
	maxPayloadSize = envInt("MATCHER_MCP_MAX_PAYLOAD_SIZE", 1<<20) // 1 MiB
	evalTimeout    = envDuration("MATCHER_MCP_EVAL_TIMEOUT", 30*time.Second)
)

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func envDuration(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}

// MatchInput is the input schema for the match tool.
type MatchInput struct {
	Query string `json:"query" jsonschema:"matcher query to evaluate (see tool description for grammar)"`
	Data  any    `json:"data" jsonschema:"JSON object whose fields are matched by the query"`
}

// MatchResult is the structured output of the match tool.
type MatchResult struct {
	Matched bool   `json:"matched"`
	Error   string `json:"error,omitempty"`
}

// ValidateInput is the input schema for the validate_query tool.
type ValidateInput struct {
	Query string `json:"query" jsonschema:"matcher query to check for syntax errors"`
}

// ValidateResult is the structured output of the validate_query tool.
type ValidateResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

const matchDescription = `Evaluate a matcher query against a JSON object and report whether it matches.

Query grammar: conditions are <field> <operator> <value>.
Operators: =, !=, <>, >, >=, <, <= (strings compare lexicographically; numeric strings coerce to numbers).
Combine conditions with AND, OR, NOT; use parentheses for grouping.
Membership: <field> IN (v1, v2, ...). Substring / array-element check: <field> CONTAINS <value>.
Values: numbers (42, -1.5, 1e3), strings ('single' or "double" quoted), TRUE, FALSE, NULL,
regex literals /pattern/ with optional 'i' flag (e.g. /error/i).
Examples:
  age >= 18 AND country = 'JP'
  status IN ('active', 'trial') OR email = /.*@example\.com$/
  NOT (role = 'admin' AND deleted = TRUE)
  tags CONTAINS 'go'
Returns {"matched": bool, "error": string}. If error is non-empty, fix the
query or data and retry; matched is false on errors.`

const validateDescription = `Check a matcher query for syntax errors without evaluating it.
Returns {"ok": true} when the query parses, or {"ok": false, "error": string}
describing the syntax problem. Use this before 'match' when unsure of the syntax.
Grammar: <field> <op> <value> with ops =, !=, <>, >, >=, <, <=, IN (...), CONTAINS;
AND/OR/NOT; parentheses; values are numbers, 'strings', TRUE/FALSE, NULL, /regex/i.`

// handleMatch implements the match tool: compile the query, coerce data into a
// matcher.Context, and evaluate. Parse and evaluation failures are reported in
// the result payload so callers can fix their input and retry.
func handleMatch(ctx context.Context, query string, data any) MatchResult {
	if len(query) > maxQueryLength {
		return MatchResult{Error: fmt.Sprintf("query too long: %d bytes (max %d)", len(query), maxQueryLength)}
	}

	// A new Matcher (and participle parser) per call keeps concurrent tool
	// calls independent; parsing is cheap relative to JSON handling.
	m, err := matcher.NewMatcher(query)
	if err != nil {
		return MatchResult{Error: err.Error()}
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return MatchResult{Error: fmt.Sprintf("data is not JSON-serializable: %v", err)}
	}
	if len(dataJSON) > maxPayloadSize {
		return MatchResult{Error: fmt.Sprintf("data payload too large: %d bytes (max %d)", len(dataJSON), maxPayloadSize)}
	}

	c := matcher.Context{}
	if err := json.Unmarshal(dataJSON, &c); err != nil || c == nil {
		return MatchResult{Error: "data must be a JSON object"}
	}

	evalCtx, cancel := context.WithTimeout(ctx, evalTimeout)
	defer cancel()

	matched, err := m.TestWithContext(evalCtx, &c)
	if err != nil {
		return MatchResult{Error: err.Error()}
	}
	return MatchResult{Matched: matched}
}

// handleValidateQuery implements the validate_query tool: parse check only.
func handleValidateQuery(query string) ValidateResult {
	if len(query) > maxQueryLength {
		return ValidateResult{Error: fmt.Sprintf("query too long: %d bytes (max %d)", len(query), maxQueryLength)}
	}
	if _, err := matcher.NewMatcher(query); err != nil {
		return ValidateResult{Error: err.Error()}
	}
	return ValidateResult{OK: true}
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "matcher-mcp", Version: "0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "match",
		Description: matchDescription,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in MatchInput) (*mcp.CallToolResult, MatchResult, error) {
		return nil, handleMatch(ctx, in.Query, in.Data), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate_query",
		Description: validateDescription,
	}, func(_ context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, ValidateResult, error) {
		return nil, handleValidateQuery(in.Query), nil
	})

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("matcher-mcp: %v", err)
	}
}

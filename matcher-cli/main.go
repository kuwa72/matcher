package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"

	"github.com/kuwa72/matcher"
)

// Exit codes (grep convention):
//   0 - matched
//   1 - unmatched
//   2 - error
const (
	exitMatched   = 0
	exitUnmatched = 1
	exitError     = 2
)

// CLI defines the command-line interface structure
var (
	cli struct {
		QUERY   string `arg:"" required:"" help:"QUERY to parse."`
		Debug   bool   `help:"Enable debug mode" default:"false"`
		Timeout int    `help:"Timeout in seconds" default:"30"`
	}
)

func main() {
	// Parse command line arguments
	kong.Parse(&cli)

	os.Exit(run(cli.QUERY, cli.Debug, cli.Timeout, os.Stdin, os.Stdout, os.Stderr))
}

func run(query string, debug bool, timeout int, stdin io.Reader, stdout, stderr io.Writer) int {
	// Create a context with timeout and signal handling
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create a new matcher
	m, err := matcher.NewMatcher(query)
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing query: %v\n", err)
		return exitError
	}

	// Enable debug mode if requested
	if debug {
		m.Debug = true
	}

	// Read JSON from stdin
	j, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading input: %v\n", err)
		return exitError
	}

	// Parse JSON into context
	c := matcher.Context(make(map[string]any))
	if err := json.Unmarshal(j, &c); err != nil {
		fmt.Fprintf(stderr, "Error parsing JSON: %v\n", err)
		return exitError
	}

	// Evaluate the matcher with context
	b, err := m.TestWithContext(ctx, &c)
	if err != nil {
		fmt.Fprintf(stderr, "Error during evaluation: %v\n", err)
		return exitError
	}

	// Output results
	if debug {
		fmt.Fprintf(stdout, "QUERY: %#v\n", query)
		fmt.Fprintf(stdout, "JSON structure: %#v\n", c)
	}

	// Return appropriate exit code based on match result
	if b {
		fmt.Fprintln(stdout, "matched")
		return exitMatched
	}
	fmt.Fprintln(stdout, "unmatched")
	return exitUnmatched
}

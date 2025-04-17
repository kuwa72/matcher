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
	kongCtx := kong.Parse(&cli)

	// Create a context with timeout and signal handling
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cli.Timeout)*time.Second)
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Create a new matcher
	m, err := matcher.NewMatcher(cli.QUERY)
	if err != nil {
		kongCtx.FatalIfErrorf(err)
	}

	// Enable debug mode if requested
	if cli.Debug {
		m.Debug = true
	}

	// Read JSON from stdin
	j, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON into context
	c := matcher.Context(make(map[string]interface{}))
	if err := json.Unmarshal(j, &c); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Evaluate the matcher with context
	b, err := m.TestWithContext(ctx, &c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during evaluation: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if cli.Debug {
		fmt.Printf("QUERY: %#v\n", cli.QUERY)
		fmt.Printf("JSON structure: %#v\n", c)
	}

	// Return appropriate exit code based on match result
	if b {
		fmt.Println("matched")
		os.Exit(0)
	} else {
		fmt.Println("unmatched")
		os.Exit(1)
	}
}

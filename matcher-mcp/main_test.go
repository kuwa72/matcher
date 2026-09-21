package main

import (
	"context"
	"strings"
	"testing"
)

func TestHandleMatch(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		data        any
		wantMatched bool
		wantErr     string // substring; empty means no error expected
	}{
		{
			name:        "matching equality",
			query:       "country = 'JP'",
			data:        map[string]any{"country": "JP", "age": 30},
			wantMatched: true,
		},
		{
			name:        "matching compound",
			query:       "age >= 18 AND status IN ('active', 'trial')",
			data:        map[string]any{"age": 20, "status": "active"},
			wantMatched: true,
		},
		{
			name:        "matching regex",
			query:       "email = /.*@example\\.com$/",
			data:        map[string]any{"email": "a@example.com"},
			wantMatched: true,
		},
		{
			name:        "non-matching",
			query:       "country = 'US'",
			data:        map[string]any{"country": "JP"},
			wantMatched: false,
		},
		{
			name:        "missing field does not match",
			query:       "age > 10",
			data:        map[string]any{"name": "x"},
			wantMatched: false,
		},
		{
			name:    "bad query reports error in payload",
			query:   "age =>= 18",
			data:    map[string]any{"age": 20},
			wantErr: "parse error",
		},
		{
			name:    "empty query reports error",
			query:   "",
			data:    map[string]any{},
			wantErr: "empty query",
		},
		{
			name:    "non-object data (array) reports error",
			query:   "a = 1",
			data:    []any{1, 2, 3},
			wantErr: "JSON object",
		},
		{
			name:    "non-object data (scalar) reports error",
			query:   "a = 1",
			data:    "hello",
			wantErr: "JSON object",
		},
		{
			name:    "null data reports error",
			query:   "a = 1",
			data:    nil,
			wantErr: "JSON object",
		},
		{
			name:    "eval error reported in payload",
			query:   "a CONTAINS 'x'",
			data:    map[string]any{"a": 42},
			wantErr: "CONTAINS",
		},
		{
			name:    "oversized query reports error",
			query:   strings.Repeat("a = 1 OR ", maxQueryLength/9+1) + "a = 1",
			data:    map[string]any{"a": 1},
			wantErr: "query too long",
		},
		{
			name:    "oversized data reports error",
			query:   "a = 1",
			data:    map[string]any{"a": 1, "pad": strings.Repeat("x", maxPayloadSize)},
			wantErr: "too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleMatch(ctx, tt.query, tt.data)
			if tt.wantErr != "" {
				if got.Error == "" {
					t.Fatalf("handleMatch() error = \"\", want substring %q", tt.wantErr)
				}
				if !strings.Contains(got.Error, tt.wantErr) {
					t.Fatalf("handleMatch() error = %q, want substring %q", got.Error, tt.wantErr)
				}
				if got.Matched {
					t.Fatalf("handleMatch() matched = true on error, want false")
				}
				return
			}
			if got.Error != "" {
				t.Fatalf("handleMatch() unexpected error = %q", got.Error)
			}
			if got.Matched != tt.wantMatched {
				t.Fatalf("handleMatch() matched = %v, want %v", got.Matched, tt.wantMatched)
			}
		})
	}
}

func TestHandleValidateQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantOK  bool
		wantErr string
	}{
		{name: "valid query", query: "a = 1 AND b = 'x'", wantOK: true},
		{name: "valid IN query", query: "a IN (1, 2, 3)", wantOK: true},
		{name: "invalid syntax", query: "a = = 1", wantOK: false, wantErr: "parse error"},
		{name: "empty query", query: "", wantOK: false, wantErr: "empty query"},
		{
			name:    "oversized query",
			query:   strings.Repeat("x", maxQueryLength+1),
			wantOK:  false,
			wantErr: "query too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleValidateQuery(tt.query)
			if got.OK != tt.wantOK {
				t.Fatalf("handleValidateQuery() ok = %v, want %v (error %q)", got.OK, tt.wantOK, got.Error)
			}
			if tt.wantErr != "" && !strings.Contains(got.Error, tt.wantErr) {
				t.Fatalf("handleValidateQuery() error = %q, want substring %q", got.Error, tt.wantErr)
			}
		})
	}
}

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		stdin          string
		debug          bool
		wantExitCode   int
		wantStdout     string
		wantStderrPart string
	}{
		{
			name:         "match returns 0",
			query:        `age = 40`,
			stdin:        `{"age": 40}`,
			wantExitCode: exitMatched,
			wantStdout:   "matched\n",
		},
		{
			name:         "unmatched returns 1",
			query:        `age = 40`,
			stdin:        `{"age": 30}`,
			wantExitCode: exitUnmatched,
			wantStdout:   "unmatched\n",
		},
		{
			name:           "invalid JSON returns 2",
			query:          `age = 40`,
			stdin:          `{not json`,
			wantExitCode:   exitError,
			wantStderrPart: "Error parsing JSON",
		},
		{
			name:           "invalid query returns 2",
			query:          `age =`,
			stdin:          `{"age": 40}`,
			wantExitCode:   exitError,
			wantStderrPart: "Error parsing query",
		},
		{
			name:         "debug match returns 0",
			query:        `age = 40`,
			stdin:        `{"age": 40}`,
			debug:        true,
			wantExitCode: exitMatched,
			wantStdout:   "matched\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.query, tt.debug, 30, strings.NewReader(tt.stdin), &stdout, &stderr)

			assert.Equal(t, tt.wantExitCode, code)
			if tt.wantStdout != "" {
				assert.Contains(t, stdout.String(), tt.wantStdout)
			}
			if tt.wantStderrPart != "" {
				assert.Contains(t, stderr.String(), tt.wantStderrPart)
			}
		})
	}
}

package util

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		input    string
		expected string
		warn     bool
	}{
		{"Hello ${TEST_VAR}", "Hello test_value", false},
		{"Hello $TEST_VAR", "Hello test_value", false},
		{"Hello ${MISSING_VAR}", "Hello ", true},
		{"No vars here", "No vars here", false},
	}

	for _, tt := range tests {
		// Capture stderr
		r, w, _ := os.Pipe()
		oldStderr := os.Stderr
		os.Stderr = w

		result, warned := ExpandEnv(tt.input)

		w.Close()
		os.Stderr = oldStderr
		var buf bytes.Buffer
		io.Copy(&buf, r)
		stderrOutput := buf.String()

		if result != tt.expected {
			t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
		}

		if warned != tt.warn {
			t.Errorf("ExpandEnv(%q) warned = %v, want %v", tt.input, warned, tt.warn)
		}

		if tt.warn {
			if !strings.Contains(stderrOutput, "Warning: environment variable") {
				t.Errorf("ExpandEnv(%q) expected warning in stderr, got none", tt.input)
			}
		} else {
			if stderrOutput != "" {
				t.Errorf("ExpandEnv(%q) unexpected warning in stderr: %s", tt.input, stderrOutput)
			}
		}
	}
}

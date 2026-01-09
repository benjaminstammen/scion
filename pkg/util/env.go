package util

import (
	"fmt"
	"os"
)

// ExpandEnv replaces ${var} or $var in the string according to the values
// of the current environment variables. It warns to stderr if a variable is unset.
// It returns the expanded string and a boolean indicating if any warning was printed.
func ExpandEnv(s string) (string, bool) {
	warned := false
	expanded := os.Expand(s, func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: environment variable %q is not set\n", key)
			warned = true
			return ""
		}
		return val
	})
	return expanded, warned
}

// FirstNonEmpty returns the first non-empty string from the given slice.
func FirstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}


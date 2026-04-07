package main

import "testing"

/**
 * Test Plan: urlUnsafeChars regex
 *
 * Scenario: Remove characters unsafe for URLs
 *   Given various input strings with unsafe characters
 *   When urlUnsafeChars.ReplaceAllString is applied
 *   Then only alphanumeric characters, hyphens, and underscores remain
 */

func TestURLUnsafeChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "spaces removed",
			input: "Hello World",
			want:  "HelloWorld",
		},
		{
			name:  "hyphens and underscores kept",
			input: "test-file_name",
			want:  "test-file_name",
		},
		{
			name:  "non-ASCII removed",
			input: "café",
			want:  "caf",
		},
		{
			name:  "path separators removed",
			input: "a/b\\c:d",
			want:  "abcd",
		},
		{
			name:  "empty string unchanged",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlUnsafeChars.ReplaceAllString(tt.input, "")
			if got != tt.want {
				t.Errorf("urlUnsafeChars.ReplaceAllString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

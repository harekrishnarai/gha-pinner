package main

import "testing"

func TestExtractRepoNameFromURL_ValidFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"owner/repo", "owner/repo"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"https://github.com/owner/repo.git", "owner/repo"},
		{"git@github.com:owner/repo.git", "owner/repo"},
		{"https://x-access-token:abc123@github.com/owner/private-repo.git", "owner/private-repo"},
	}

	for _, tc := range tests {
		got, err := extractRepoNameFromURL(tc.input)
		if err != nil {
			t.Fatalf("expected no error for %q, got: %v", tc.input, err)
		}
		if got != tc.expected {
			t.Fatalf("unexpected repo name for %q: got=%q expected=%q", tc.input, got, tc.expected)
		}
	}
}

func TestExtractRepoNameFromURL_InvalidFormats(t *testing.T) {
	tests := []string{
		"https://github.com/owner",
		"not-a-repo",
		"https://example.com/owner/repo",
	}

	for _, input := range tests {
		if _, err := extractRepoNameFromURL(input); err == nil {
			t.Fatalf("expected error for invalid input %q", input)
		}
	}
}

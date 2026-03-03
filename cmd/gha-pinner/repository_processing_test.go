package main

import "testing"

func TestProcessRepositoryNames_Empty(t *testing.T) {
	success, failed := processRepositoryNames([]string{})
	if success != 0 || failed != 0 {
		t.Fatalf("expected zero results for empty input, got success=%d failed=%d", success, failed)
	}
}

func TestGetPRTitleForRepository(t *testing.T) {
	tests := []struct {
		repo     string
		expected string
	}{
		{"ossf/scorecard", ":seedling: security: pin GitHub Actions to commit hashes"},
		{"kubernetes/kubernetes", ":seedling: security: pin GitHub Actions to commit hashes"},
		{"owner/repo", "security: pin GitHub Actions to commit hashes"},
	}

	for _, tc := range tests {
		got := getPRTitleForRepository(tc.repo)
		if got != tc.expected {
			t.Fatalf("unexpected title for %s: got=%q expected=%q", tc.repo, got, tc.expected)
		}
	}
}

func TestGetPRSearchPattern(t *testing.T) {
	tests := []struct {
		repo     string
		expected string
	}{
		{"ossf/allstar", ":seedling: security: pin GitHub Actions to commit hashes in:title"},
		{"k8s.io/some-repo", ":seedling: security: pin GitHub Actions to commit hashes in:title"},
		{"owner/repo", "security: pin GitHub Actions to commit hashes in:title"},
	}

	for _, tc := range tests {
		got := getPRSearchPattern(tc.repo)
		if got != tc.expected {
			t.Fatalf("unexpected search pattern for %s: got=%q expected=%q", tc.repo, got, tc.expected)
		}
	}
}

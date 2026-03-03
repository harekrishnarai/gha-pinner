package main

import (
	"strings"
	"testing"
)

func saveAuthGlobals() (string, string, int) {
	return authMode, githubToken, repoWorkers
}

func restoreAuthGlobals(mode, token string, workers int) {
	authMode = mode
	githubToken = token
	repoWorkers = workers
}

func TestGetAuthenticatedCloneURL_PATMode(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
	})

	authMode = "pat"
	githubToken = "test-token-123"

	cloneURL, err := getAuthenticatedCloneURL("octo-org/octo-repo")
	if err != nil {
		t.Fatalf("expected clone URL without error, got: %v", err)
	}

	if !strings.Contains(cloneURL, "github.com/octo-org/octo-repo.git") {
		t.Fatalf("expected clone URL to contain repository path, got: %s", cloneURL)
	}
	if !strings.Contains(cloneURL, "x-access-token:test-token-123") {
		t.Fatalf("expected clone URL to include PAT credentials, got: %s", cloneURL)
	}
}

func TestGetAuthenticatedCloneURL_PATModeMissingToken(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
	})

	authMode = "pat"
	githubToken = ""

	_, err := getAuthenticatedCloneURL("octo-org/octo-repo")
	if err == nil {
		t.Fatal("expected error when PAT token is missing")
	}
	if !strings.Contains(err.Error(), "missing GitHub token") {
		t.Fatalf("expected missing token error, got: %v", err)
	}
}

func TestGetAuthenticatedCloneURL_GHMode(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
	})

	authMode = "gh"
	githubToken = "ignored-in-gh-mode"

	_, err := getAuthenticatedCloneURL("octo-org/octo-repo")
	if err == nil {
		t.Fatal("expected error in gh mode")
	}
	if !strings.Contains(err.Error(), "pat mode") {
		t.Fatalf("expected pat-mode specific error, got: %v", err)
	}
}

func TestCreatePullRequestFallback_PATMode(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
	})

	authMode = "pat"

	result := createPullRequestFallback("owner/repo", "title", "body", true, "")
	if result.ExitCode == 0 {
		t.Fatal("expected fallback to fail in pat mode")
	}
	if !strings.Contains(result.Stderr, "failed to create pull request") {
		t.Fatalf("unexpected fallback error: %s", result.Stderr)
	}
}

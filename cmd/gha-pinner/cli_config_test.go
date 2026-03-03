package main

import (
	"os"
	"testing"
)

func TestValidateRuntimeConfig_InvalidAuthMode(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
	})

	authMode = "invalid"
	repoWorkers = 2
	githubToken = ""

	if err := validateRuntimeConfig(); err == nil {
		t.Fatal("expected validation error for invalid auth mode")
	}
}

func TestValidateRuntimeConfig_PATRequiresToken(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	oldGH := getenvOrEmpty("GH_TOKEN")
	oldGitHubToken := getenvOrEmpty("GITHUB_TOKEN")
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
		_ = os.Setenv("GH_TOKEN", oldGH)
		_ = os.Setenv("GITHUB_TOKEN", oldGitHubToken)
	})

	_ = os.Unsetenv("GH_TOKEN")
	_ = os.Unsetenv("GITHUB_TOKEN")

	authMode = "pat"
	repoWorkers = 2
	githubToken = ""

	if err := validateRuntimeConfig(); err == nil {
		t.Fatal("expected validation error when PAT token is missing")
	}
}

func TestValidateRuntimeConfig_RepoWorkersValidation(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
	})

	authMode = "gh"
	repoWorkers = 0

	if err := validateRuntimeConfig(); err == nil {
		t.Fatal("expected validation error for repo-workers < 1")
	}
}

func getenvOrEmpty(key string) string {
	v, _ := os.LookupEnv(key)
	return v
}

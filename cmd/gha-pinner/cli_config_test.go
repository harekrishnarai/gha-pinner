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

func saveHardeningGlobals() (bool, string, bool, []string, map[string]string) {
	cp := make(map[string]string, len(runnerMap))
	for k, v := range runnerMap {
		cp[k] = v
	}
	return injectHardenRunner, egressPolicy, pinRunners, runnerMapRaw, cp
}

func restoreHardeningGlobals(inject bool, policy string, pinR bool, mapRaw []string, mapParsed map[string]string) {
	injectHardenRunner = inject
	egressPolicy = policy
	pinRunners = pinR
	runnerMapRaw = mapRaw
	runnerMap = mapParsed
}

func TestValidateRuntimeConfig_InvalidEgressPolicy(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	oldInject, oldPolicy, oldPin, oldMap, oldRunnerMap := saveHardeningGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
		restoreHardeningGlobals(oldInject, oldPolicy, oldPin, oldMap, oldRunnerMap)
	})

	authMode = "gh"
	repoWorkers = 2
	injectHardenRunner = true
	egressPolicy = "dangerous"

	if err := validateRuntimeConfig(); err == nil {
		t.Fatal("expected validation error for invalid egress-policy")
	}
}

func TestValidateRuntimeConfig_ValidEgressPolicies(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	oldInject, oldPolicy, oldPin, oldMap, oldRunnerMap := saveHardeningGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
		restoreHardeningGlobals(oldInject, oldPolicy, oldPin, oldMap, oldRunnerMap)
	})

	authMode = "gh"
	repoWorkers = 2
	injectHardenRunner = true

	for _, policy := range []string{"audit", "block"} {
		egressPolicy = policy
		if err := validateRuntimeConfig(); err != nil {
			t.Fatalf("expected no error for egress-policy=%q, got: %v", policy, err)
		}
	}
}

func TestValidateRuntimeConfig_EgressPolicyIgnoredWithoutInjectFlag(t *testing.T) {
	oldMode, oldToken, oldWorkers := saveAuthGlobals()
	oldInject, oldPolicy, oldPin, oldMap, oldRunnerMap := saveHardeningGlobals()
	t.Cleanup(func() {
		restoreAuthGlobals(oldMode, oldToken, oldWorkers)
		restoreHardeningGlobals(oldInject, oldPolicy, oldPin, oldMap, oldRunnerMap)
	})

	authMode = "gh"
	repoWorkers = 2
	injectHardenRunner = false
	egressPolicy = "dangerous"

	if err := validateRuntimeConfig(); err != nil {
		t.Fatalf("expected no error when --inject-harden-runner is false, got: %v", err)
	}
}

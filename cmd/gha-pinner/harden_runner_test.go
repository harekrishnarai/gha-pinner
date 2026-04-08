package main

import (
	"strings"
	"testing"
)

func TestBuildHardenRunnerBlock(t *testing.T) {
	block := buildHardenRunnerBlock("    ", "abc123def456abc123def456abc123def456abc1", "v2.16.0", "audit")

	if !strings.Contains(block, "    - name: Harden the runner") {
		t.Errorf("block missing step name with correct indent:\n%s", block)
	}
	if !strings.Contains(block, "step-security/harden-runner@abc123def456abc123def456abc123def456abc1 # v2.16.0") {
		t.Errorf("block missing pinned uses line:\n%s", block)
	}
	if !strings.Contains(block, "egress-policy: audit") {
		t.Errorf("block missing egress-policy:\n%s", block)
	}
}

func TestInjectHardenRunnerStep_BasicInjection(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
`
	sha := "abc123def456abc123def456abc123def456abc1"
	version := "v2.16.0"
	updated, count := injectHardenRunnerStep(content, sha, version, "audit")

	if count != 1 {
		t.Errorf("expected 1 injection, got %d", count)
	}
	if !strings.Contains(updated, "step-security/harden-runner@"+sha) {
		t.Errorf("expected harden-runner in output:\n%s", updated)
	}
	hardenIdx := strings.Index(updated, "step-security/harden-runner")
	checkoutIdx := strings.Index(updated, "actions/checkout")
	if hardenIdx > checkoutIdx {
		t.Errorf("harden-runner must be before checkout, but got indices harden=%d checkout=%d", hardenIdx, checkoutIdx)
	}
}

func TestInjectHardenRunnerStep_AlreadyPresent_Skipped(t *testing.T) {
	sha := "abc123def456abc123def456abc123def456abc1"
	content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Harden Runner
        uses: step-security/harden-runner@` + sha + ` # v2.16.0
        with:
          egress-policy: audit
      - name: Checkout
        uses: actions/checkout@v4
`
	_, count := injectHardenRunnerStep(content, sha, "v2.16.0", "audit")
	if count != 0 {
		t.Errorf("expected 0 injections when harden-runner already present, got %d", count)
	}
}

func TestInjectHardenRunnerStep_MultipleJobs(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-node@v4
`
	sha := "abc123def456abc123def456abc123def456abc1"
	updated, count := injectHardenRunnerStep(content, sha, "v2.16.0", "block")

	if count != 2 {
		t.Errorf("expected 2 injections for 2 jobs, got %d", count)
	}
	occurrences := strings.Count(updated, "step-security/harden-runner")
	if occurrences != 2 {
		t.Errorf("expected 2 harden-runner occurrences, got %d", occurrences)
	}
}

func TestInjectHardenRunnerStep_EgressPolicyRespected(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`
	sha := "abc123def456abc123def456abc123def456abc1"
	updated, _ := injectHardenRunnerStep(content, sha, "v2.16.0", "block")

	if !strings.Contains(updated, "egress-policy: block") {
		t.Errorf("expected egress-policy: block in output:\n%s", updated)
	}
}

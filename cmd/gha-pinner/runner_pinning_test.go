package main

import (
	"strings"
	"testing"
)

func TestResolveRunnerLabel_CustomMapFirst(t *testing.T) {
	custom := map[string]string{"ubuntu-latest": "ubuntu-22.04"}
	got := resolveRunnerLabel("ubuntu-latest", custom)
	if got != "ubuntu-22.04" {
		t.Errorf("expected custom map value ubuntu-22.04, got %s", got)
	}
}

func TestResolveRunnerLabel_DefaultFallback(t *testing.T) {
	got := resolveRunnerLabel("ubuntu-latest", nil)
	if got != "ubuntu-24.04" {
		t.Errorf("expected default ubuntu-24.04, got %s", got)
	}
}

func TestResolveRunnerLabel_WindowsDefault(t *testing.T) {
	got := resolveRunnerLabel("windows-latest", nil)
	if got != "windows-2022" {
		t.Errorf("expected default windows-2022, got %s", got)
	}
}

func TestResolveRunnerLabel_UnknownLabelUnchanged(t *testing.T) {
	got := resolveRunnerLabel("self-hosted", nil)
	if got != "self-hosted" {
		t.Errorf("unknown label should be returned unchanged, got %s", got)
	}
}

func TestPinRunnersPass_BasicReplacement(t *testing.T) {
	content := `jobs:
  build:
    runs-on: ubuntu-latest
    steps: []
`
	updated, count := pinRunnersPass(content, nil)
	if count != 1 {
		t.Errorf("expected 1 replacement, got %d", count)
	}
	if !strings.Contains(updated, "runs-on: ubuntu-24.04") {
		t.Errorf("expected ubuntu-24.04 in output:\n%s", updated)
	}
	if strings.Contains(updated, "runs-on: ubuntu-latest") {
		t.Errorf("expected ubuntu-latest to be replaced:\n%s", updated)
	}
}

func TestPinRunnersPass_MatrixExpressionSkipped(t *testing.T) {
	content := `jobs:
  build:
    runs-on: ${{ matrix.os }}
    steps: []
`
	updated, count := pinRunnersPass(content, nil)
	if count != 0 {
		t.Errorf("expected 0 replacements for matrix expression, got %d", count)
	}
	if updated != content {
		t.Errorf("content should be unchanged for matrix expression")
	}
}

func TestPinRunnersPass_AlreadyVersioned_Unchanged(t *testing.T) {
	content := `jobs:
  build:
    runs-on: ubuntu-24.04
    steps: []
`
	updated, count := pinRunnersPass(content, nil)
	if count != 0 {
		t.Errorf("expected 0 replacements for already-versioned label, got %d", count)
	}
	if updated != content {
		t.Errorf("content should be unchanged for already-versioned label")
	}
}

func TestPinRunnersPass_MultipleJobs(t *testing.T) {
	content := `jobs:
  build:
    runs-on: ubuntu-latest
    steps: []
  test:
    runs-on: macos-latest
    steps: []
`
	updated, count := pinRunnersPass(content, nil)
	if count != 2 {
		t.Errorf("expected 2 replacements, got %d", count)
	}
	if !strings.Contains(updated, "runs-on: ubuntu-24.04") {
		t.Errorf("expected ubuntu-24.04:\n%s", updated)
	}
	if !strings.Contains(updated, "runs-on: macos-15") {
		t.Errorf("expected macos-15:\n%s", updated)
	}
}

func TestPinRunnersPass_CustomMap(t *testing.T) {
	content := `jobs:
  build:
    runs-on: ubuntu-latest
    steps: []
`
	custom := map[string]string{"ubuntu-latest": "ubuntu-22.04"}
	updated, count := pinRunnersPass(content, custom)
	if count != 1 {
		t.Errorf("expected 1 replacement, got %d", count)
	}
	if !strings.Contains(updated, "runs-on: ubuntu-22.04") {
		t.Errorf("expected custom map value ubuntu-22.04:\n%s", updated)
	}
}

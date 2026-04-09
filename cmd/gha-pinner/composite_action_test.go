package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchFile_CompositeAction_PinsUsesReferences(t *testing.T) {
	tempDir := t.TempDir()
	content := `name: My Composite Action
description: Does something
runs:
  using: composite
  steps:
    - name: Checkout
      uses: actions/checkout@v4
    - name: Setup
      uses: actions/setup-node@v4
      with:
        node-version: '20'
`
	path := filepath.Join(tempDir, "action.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// All features on — composite should only pin actions, not inject harden-runner or pin runners
	p := &WorkflowPatcher{
		injectHardenRunner: true,
		egressPolicy:       "audit",
		pinRunners:         true,
		runnerMap:          map[string]string{"ubuntu-latest": "ubuntu-24.04"},
	}
	res, err := p.patchFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.hardenInjected != 0 {
		t.Errorf("expected hardenInjected=0 for composite action, got %d", res.hardenInjected)
	}
	if res.runnersReplaced != 0 {
		t.Errorf("expected runnersReplaced=0 for composite action, got %d", res.runnersReplaced)
	}
	if res.totalActions != 2 {
		t.Errorf("expected totalActions=2, got %d", res.totalActions)
	}
}

func TestPatchFile_CompositeAction_SkipsNonActionFile(t *testing.T) {
	tempDir := t.TempDir()
	content := `foo: bar
baz: qux
`
	path := filepath.Join(tempDir, "random.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := &WorkflowPatcher{injectHardenRunner: false, egressPolicy: "audit", pinRunners: false}
	res, err := p.patchFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.totalActions != 0 {
		t.Errorf("expected totalActions=0 for non-action file, got %d", res.totalActions)
	}
}

func TestPatchLocalRepository_ScansActionsDirectory(t *testing.T) {
	repoDir := t.TempDir()
	workflowsDir := filepath.Join(repoDir, ".github", "workflows")
	actionsDir := filepath.Join(repoDir, ".github", "actions", "my-action")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(actionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	wf := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "ci.yml"), []byte(wf), 0644); err != nil {
		t.Fatal(err)
	}

	ca := `name: My Action
runs:
  using: composite
  steps:
    - uses: actions/setup-node@b4ffde65f46336ab88eb53be808477a3936bae12 # v4
`
	if err := os.WriteFile(filepath.Join(actionsDir, "action.yml"), []byte(ca), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchLocalRepository(repoDir); err != nil {
		t.Fatalf("patchLocalRepository returned unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(actionsDir, "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "actions/setup-node@b4ffde65f46336ab88eb53be808477a3936bae12") {
		t.Error("composite action file was unexpectedly modified")
	}
}

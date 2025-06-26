package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseActionReference(t *testing.T) {
	tests := []struct {
		input           string
		expectedAction  string
		expectedVersion string
		expectError     bool
	}{
		{"actions/checkout@v3", "actions/checkout", "v3", false},
		{"actions/setup-node@v4", "actions/setup-node", "v4", false},
		{"docker/build-push-action@v5", "docker/build-push-action", "v5", false},
		{"invalid-format", "", "", true},
		{"", "", "", true},
	}

	for _, test := range tests {
		action, version, err := parseActionReference(test.input)

		if test.expectError {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", test.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("Unexpected error for input %s: %v", test.input, err)
			continue
		}

		if action != test.expectedAction {
			t.Errorf("Expected action %s, got %s for input %s", test.expectedAction, action, test.input)
		}

		if version != test.expectedVersion {
			t.Errorf("Expected version %s, got %s for input %s", test.expectedVersion, version, test.input)
		}
	}
}

func TestShouldSkipAction(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"./local-action", true},
		{"actions/checkout@v3", false},
		{"actions/checkout@abc123def456789012345678901234567890abcd", true}, // commit hash
		{"normal/action@v2", false},
	}

	for _, test := range tests {
		result := shouldSkipAction(test.input)
		if result != test.expected {
			t.Errorf("Expected %v for input %s, got %v", test.expected, test.input, result)
		}
	}
}

func TestGetTempDir(t *testing.T) {
	result := getTempDir("test")

	if !strings.Contains(result, "test") {
		t.Errorf("Expected temp directory to contain 'test', got %s", result)
	}

	// Check that it's an absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("Expected absolute path, got %s", result)
	}
}

func TestGeneratePRBody(t *testing.T) {
	body := prBody

	expectedContains := []string{
		"Pin GitHub Actions",
		"commit hashes",
		"security",
		"reproducible builds",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(body, expected) {
			t.Errorf("Expected PR body to contain '%s'", expected)
		}
	}

	if len(body) < 100 {
		t.Error("PR body seems too short")
	}
}

func TestExecCommand(t *testing.T) {
	// Test a simple command that should work on Windows
	var result ExecResult
	if runtime.GOOS == "windows" {
		result = execCommand("cmd", "/c", "echo test")
	} else {
		result = execCommand("echo", "test")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	expectedOutput := "test"
	if !strings.Contains(result.Stdout, expectedOutput) {
		t.Errorf("Expected stdout to contain '%s', got '%s'", expectedOutput, result.Stdout)
	}
}

func TestExecCommandWithDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "gha-pinner-test")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Test command execution in a specific directory
	var result ExecResult
	if runtime.GOOS == "windows" {
		result = execCommandWithDir(tempDir, "cmd", "/c", "cd")
	} else {
		result = execCommandWithDir(tempDir, "pwd")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// The output should contain the temp directory path
	if !strings.Contains(result.Stdout, "gha-pinner-test") {
		t.Errorf("Expected stdout to contain temp directory path, got '%s'", result.Stdout)
	}
}

// Mock tests for functions that require external dependencies
func TestProcessWorkflowFileStructure(t *testing.T) {
	// Create a temporary workflow file for testing
	tempDir := filepath.Join(os.TempDir(), "gha-pinner-test-workflow")
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	os.MkdirAll(workflowsDir, 0755)
	defer os.RemoveAll(tempDir)

	workflowContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '18'
`

	workflowFile := filepath.Join(workflowsDir, "test.yml")
	err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Test that the file exists and can be read
	if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
		t.Error("Test workflow file was not created")
	}

	// Read the content back to verify
	content, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Errorf("Failed to read test workflow file: %v", err)
	}

	if !strings.Contains(string(content), "actions/checkout@v3") {
		t.Error("Test workflow file does not contain expected action reference")
	}
}

func TestCleanupFunction(t *testing.T) {
	// Create temporary directories that cleanup should remove
	testDirs := []string{
		getTempDir("actions"),
		getTempDir("repos"),
	}

	for _, dir := range testDirs {
		os.MkdirAll(dir, 0755)

		// Verify directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Failed to create test directory: %s", dir)
		}
	}

	// Run cleanup
	cleanup()

	// Verify directories are removed
	for _, dir := range testDirs {
		if _, err := os.Stat(dir); err == nil {
			t.Errorf("Directory should have been cleaned up: %s", dir)
		}
	}
}

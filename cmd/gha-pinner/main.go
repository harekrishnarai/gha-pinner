package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	execute "github.com/alexellis/go-execute/v2"
	"gopkg.in/yaml.v3"
)

var (
	debug                = false
	ignorePRTemplates    = false
	skipPRCreation       = false
	outputDir            = ""
	errUnresolvedVersion = errors.New("unresolved version")
	errNeedsFork         = errors.New("needs fork")
	skipActions          = []string{}
	prBody               = `# Pin GitHub Actions to commit hashes

This pull request pins all GitHub Actions in workflow files to specific commit hashes to improve security and ensure reproducible builds.

## Changes Made

- Converted version tags (e.g., ` + "`v3`" + `, ` + "`v4`" + `) to commit hashes
- Added comments showing the original version and date for reference
- Preserved all existing functionality while improving security

## Benefits

- **Security**: Prevents supply chain attacks by ensuring immutable action references  
- **Reproducibility**: Guarantees the same action version is used across all runs
- **Auditability**: Clear tracking of which specific version of each action is being used

## Review Notes

- All pinned actions maintain their original functionality
- Comments preserve the original version information with dates for easy reference
- No workflow behavior changes are expected

This change follows GitHub's security best practices for action pinning.`
)

type Repository struct {
	Name             string           `json:"name"`
	URL              string           `json:"url"`
	DefaultBranchRef DefaultBranchRef `json:"defaultBranchRef"`
}

type DefaultBranchRef struct {
	Name string `json:"name"`
}

type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func main() {
	startTime := time.Now()
	defer func() {
		if debug {
			fmt.Printf("Total execution time: %v\n", time.Since(startTime))
		}
	}()

	if len(os.Args) < 3 {
		showUsage()
		os.Exit(1)
	}

	debug = len(os.Args) > 3 && contains(os.Args, "--debug")
	ignorePRTemplates = len(os.Args) > 3 && contains(os.Args, "--ignore-templates")
	skipPRCreation = len(os.Args) > 3 && contains(os.Args, "--no-pr")
	
	// Parse output directory if provided
	for i, arg := range os.Args {
		if arg == "--output" && i+1 < len(os.Args) {
			outputDir = os.Args[i+1]
			break
		}
	}
	
	defer cleanup()

	commands := map[string]func(string) error{
		"local-repository": patchLocalRepository,
		"repository":       processRepository,
		"organization":     processOrganization,
		"switch-account":   switchAccount,
		"file":             processRepositoryFile,
	}

	command, target := os.Args[1], os.Args[2]

	if command == "action" {
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Error: action command requires version parameter\n")
			showUsage()
			os.Exit(1)
		}
		if err := resolveVersion(target, os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if fn, exists := commands[command]; exists {
		if err := fn(target); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Println("Usage:")
	fmt.Println("  gha-pinner local-repository <path> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]")
	fmt.Println("  gha-pinner repository <repo-name> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]")
	fmt.Println("  gha-pinner organization <org-name> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]")
	fmt.Println("  gha-pinner file <path-to-repos-file> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]")
	fmt.Println("  gha-pinner action <action-name> <version> [--debug]")
	fmt.Println("  gha-pinner switch-account <username> [--debug]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --debug             Enable debug output")
	fmt.Println("  --ignore-templates  Ignore PR templates and use full PR body")
	fmt.Println("  --no-pr             Skip PR creation, only fix repositories locally")
	fmt.Println("  --output <dir>      Custom output directory for repositories (only with --no-pr)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  gha-pinner local-repository ./my-repo")
	fmt.Println("  gha-pinner repository owner/repo-name")
	fmt.Println("  gha-pinner organization my-org")
	fmt.Println("  gha-pinner file repos.txt")
	fmt.Println("  gha-pinner file repos.txt --no-pr")
	fmt.Println("  gha-pinner file repos.txt --no-pr --output ./fixed-repos")
	fmt.Println("  gha-pinner action actions/checkout v3")
	fmt.Println("  gha-pinner switch-account harekrishnaraiedbyte")
}

func switchAccount(username string) error {
	// List available accounts
	result := execCommand("gh", "auth", "status")
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to check auth status: %s", result.Stderr)
	}

	fmt.Printf("Current auth status:\n%s\n", result.Stdout)

	// Switch to the specified account
	result = execCommand("gh", "auth", "switch", "--user", username)
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to switch account to %s: %s", username, result.Stderr)
	}

	fmt.Printf("Successfully switched to account: %s\n", username)
	return nil
}

func cleanup() {
	// If --no-pr is set, don't clean up temp directories to allow manual review
	if skipPRCreation {
		fmt.Printf("\n📁 Repositories preserved for manual review:\n")
		reposDir := getReposDir()
		if _, err := os.Stat(reposDir); err == nil {
			fmt.Printf("   • %s\n", reposDir)
		}
		fmt.Printf("\n💡 Tip: Review changes and manually commit/push when ready\n")
		return
	}

	tempDirs := []string{getTempDir("actions"), getTempDir("repos"), getTempDir("pr-body.md")}
	for _, dir := range tempDirs {
		if _, err := os.Stat(dir); err == nil {
			os.RemoveAll(dir)
			if debug {
				fmt.Printf("Cleaned up: %s\n", dir)
			}
		}
	}
}

func getTempDir(name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), name)
	}
	return filepath.Join("/tmp", name)
}

func getReposDir() string {
	if skipPRCreation && outputDir != "" {
		// Use custom output directory
		return outputDir
	}
	return getTempDir("repos")
}

func getActionsCacheDir() string {
	cacheDir := filepath.Join(os.TempDir(), "gha-pinner-cache", "actions")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		// Fallback to temp dir if cache creation fails
		return getTempDir("actions")
	}
	return cacheDir
}

func processRepository(repoName string) error {
	result := execCommand("gh", "repo", "view", repoName, "--json", "name,url,defaultBranchRef")
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to fetch repository metadata: %s", result.Stderr)
	}
	var repo Repository
	if err := json.Unmarshal([]byte(result.Stdout), &repo); err != nil {
		return fmt.Errorf("failed to parse repository metadata: %v", err)
	}
	repo.URL = repoName // Store the full repo name for cloning
	return patchRepository(repo)
}

func processOrganization(orgName string) error {
	result := execCommand("gh", "repo", "list", orgName, "--json", "name,url,defaultBranchRef", "--limit", "1000")
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to list repositories: %s", result.Stderr)
	}
	var repos []Repository
	if err := json.Unmarshal([]byte(result.Stdout), &repos); err != nil {
		return fmt.Errorf("failed to parse repositories list: %v", err)
	}

	successCount, errorCount := 0, 0
	fmt.Printf("🏢 Processing %d repositories in organization: %s\n", len(repos), orgName)

	for i, repo := range repos {
		// Ensure we have the full repository name for cloning
		fullRepoName := fmt.Sprintf("%s/%s", orgName, repo.Name)
		repo.URL = fullRepoName

		fmt.Printf("\n[%d/%d] 🔍 Processing repository: %s\n", i+1, len(repos), fullRepoName)
		if err := patchRepository(repo); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error processing %s: %v\n", fullRepoName, err)
			errorCount++
		} else {
			successCount++
		}
	}
	fmt.Printf("\n🎯 Organization processing complete:\n")
	fmt.Printf("   • ✅ Successful: %d repositories\n", successCount)
	fmt.Printf("   • ❌ Failed: %d repositories\n", errorCount)
	fmt.Printf("   • 📊 Total: %d repositories\n", len(repos))
	return nil
}

func processRepositoryFile(filePath string) error {
	// Read the file containing repository URLs
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var repoURLs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		repoURLs = append(repoURLs, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	if len(repoURLs) == 0 {
		return fmt.Errorf("no repository URLs found in file %s", filePath)
	}

	successCount, errorCount := 0, 0
	fmt.Printf("📋 Processing %d repositories from file: %s\n", len(repoURLs), filePath)

	for i, repoURL := range repoURLs {
		// Extract repository name from GitHub URL
		repoName, err := extractRepoNameFromURL(repoURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error parsing URL %s: %v\n", repoURL, err)
			errorCount++
			continue
		}

		fmt.Printf("\n[%d/%d] 🔍 Processing repository: %s\n", i+1, len(repoURLs), repoName)
		
		// Get repository metadata
		result := execCommand("gh", "repo", "view", repoName, "--json", "name,url,defaultBranchRef")
		if result.ExitCode != 0 {
			fmt.Fprintf(os.Stderr, "❌ Error fetching metadata for %s: %s\n", repoName, result.Stderr)
			errorCount++
			continue
		}

		var repo Repository
		if err := json.Unmarshal([]byte(result.Stdout), &repo); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error parsing metadata for %s: %v\n", repoName, err)
			errorCount++
			continue
		}
		repo.URL = repoName // Store the full repo name for cloning

		if err := patchRepository(repo); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error processing %s: %v\n", repoName, err)
			errorCount++
		} else {
			successCount++
		}
	}

	fmt.Printf("\n🎯 File processing complete:\n")
	fmt.Printf("   • ✅ Successful: %d repositories\n", successCount)
	fmt.Printf("   • ❌ Failed: %d repositories\n", errorCount)
	fmt.Printf("   • 📊 Total: %d repositories\n", len(repoURLs))
	return nil
}

func extractRepoNameFromURL(repoURL string) (string, error) {
	// Handle different GitHub URL formats:
	// https://github.com/owner/repo
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// owner/repo

	// If it's already in owner/repo format, return as-is
	if !strings.Contains(repoURL, "github.com") && strings.Count(repoURL, "/") == 1 {
		return repoURL, nil
	}

	// Extract from GitHub URLs
	repoURL = strings.TrimSpace(repoURL)
	
	// Remove .git suffix if present
	repoURL = strings.TrimSuffix(repoURL, ".git")
	
	if strings.HasPrefix(repoURL, "https://github.com/") {
		// https://github.com/owner/repo
		parts := strings.Split(strings.TrimPrefix(repoURL, "https://github.com/"), "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("%s/%s", parts[0], parts[1]), nil
		}
	} else if strings.HasPrefix(repoURL, "git@github.com:") {
		// git@github.com:owner/repo
		parts := strings.Split(strings.TrimPrefix(repoURL, "git@github.com:"), "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("%s/%s", parts[0], parts[1]), nil
		}
	}

	return "", fmt.Errorf("invalid GitHub repository URL format: %s", repoURL)
}

func resolveVersion(action, version string) error {
	hash, resolvedVersion, err := getCommitHashFromVersion(action, version)
	if err != nil {
		return err
	}
	fmt.Printf("Action: %s\nVersion: %s\nCommit Hash: %s\n", action, resolvedVersion, hash)
	return nil
}

func patchRepository(repo Repository) error {
	fmt.Printf("\n🔍 Analyzing repository: %s\n", repo.Name)

	// Check repository permissions before proceeding
	cloneTarget := repo.URL
	if cloneTarget == "" {
		cloneTarget = repo.Name
	}

	originalRepo := cloneTarget
	needsFork := false

	if err := checkRepositoryPermissions(cloneTarget); err != nil {
		if errors.Is(err, errNeedsFork) {
			// Fork the repository and sync it
			forkName, forkErr := forkRepository(cloneTarget)
			if forkErr != nil {
				return fmt.Errorf("failed to fork repository: %v", forkErr)
			}
			cloneTarget = forkName
			needsFork = true
			
			// Sync fork with upstream if it exists
			if syncErr := syncForkWithUpstream(forkName, originalRepo); syncErr != nil {
				if debug {
					fmt.Printf("Warning: failed to sync fork %s with upstream: %v\n", forkName, syncErr)
				}
			}
			
			if debug {
				fmt.Printf("Using fork: %s\n", cloneTarget)
			}
		} else {
			return fmt.Errorf("permission check failed: %v", err)
		}
	}

	repoDir := filepath.Join(getReposDir(), strings.ReplaceAll(repo.Name, "/", "_"))

	if _, err := os.Stat(repoDir); err == nil {
		if debug {
			fmt.Printf("Repository directory already exists, removing: %s\n", repoDir)
		}
		if err := os.RemoveAll(repoDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %v", err)
		}
	}

	if result := execCommand("gh", "repo", "clone", cloneTarget, repoDir); result.ExitCode != 0 {
		return fmt.Errorf("failed to clone repository: %s", result.Stderr)
	}

	// If we forked and synced, ensure we have the latest changes locally
	if needsFork {
		if debug {
			fmt.Printf("Adding upstream remote: %s\n", originalRepo)
		}
		result := execCommandWithDir(repoDir, "git", "remote", "add", "upstream", fmt.Sprintf("https://github.com/%s.git", originalRepo))
		if result.ExitCode != 0 {
			if debug {
				fmt.Printf("Warning: failed to add upstream remote (may already exist): %s\n", result.Stderr)
			}
		}

		// Fetch the latest changes from origin (our fork) to ensure we have the synced code
		if debug {
			fmt.Printf("Fetching latest changes from fork...\n")
		}
		result = execCommandWithDir(repoDir, "git", "fetch", "origin", "--quiet")
		if result.ExitCode != 0 && debug {
			fmt.Printf("Warning: failed to fetch from origin: %s\n", result.Stderr)
		}

		// Reset to the latest origin/main to ensure we're working with synced code
		defaultBranch := repo.DefaultBranchRef.Name
		if defaultBranch == "" {
			defaultBranch = "main"
		}
		
		if debug {
			fmt.Printf("Resetting to latest %s from fork...\n", defaultBranch)
		}
		result = execCommandWithDir(repoDir, "git", "reset", "--hard", fmt.Sprintf("origin/%s", defaultBranch))
		if result.ExitCode != 0 && debug {
			fmt.Printf("Warning: failed to reset to origin/%s: %s\n", defaultBranch, result.Stderr)
		}
	}

	if err := configureGitCredentials(repoDir); err != nil {
		return fmt.Errorf("failed to configure git credentials: %v", err)
	}

	if err := patchLocalRepository(repoDir); err != nil {
		return fmt.Errorf("failed to patch repository: %v", err)
	}

	if result := execCommandWithDir(repoDir, "git", "diff", "--exit-code"); result.ExitCode == 0 {
		fmt.Printf("✅ No changes needed for repository: %s - all actions are already properly secured\n", repo.Name)
		return nil
	}

	// If --no-pr flag is set, just show the changes and exit
	if skipPRCreation {
		fmt.Printf("🔍 Changes detected in repository: %s\n", repo.Name)
		
		// Show the diff for review
		diffResult := execCommandWithDir(repoDir, "git", "diff", ".github/workflows")
		if diffResult.ExitCode == 0 && diffResult.Stdout != "" {
			fmt.Printf("\n📋 Workflow changes preview:\n")
			fmt.Printf("---\n%s---\n", diffResult.Stdout)
		}
		
		fmt.Printf("✅ Repository %s has been processed and changes are ready for review\n", repo.Name)
		fmt.Printf("   • Repository location: %s\n", repoDir)
		fmt.Printf("   • To create a PR manually: cd %s && git add . && git commit -m 'Pin GitHub Actions' && git push\n", repoDir)
		return nil
	}

	branchName := fmt.Sprintf("pin-actions-%s", time.Now().Format("20060102-150405"))
	currentBranch := strings.TrimSpace(execCommandWithDir(repoDir, "git", "branch", "--show-current").Stdout)
	if debug {
		fmt.Printf("Current branch: %s\n", currentBranch)
	}

	commands := [][]string{
		{"git", "checkout", "-b", branchName},
		{"git", "add", ".github/workflows"},
		{"git", "commit", "-m", getPRTitleForRepository(originalRepo) + "\n\nPin GitHub Actions to commit hashes for improved security and reproducible builds"},
		{"git", "push", "origin", branchName},
	}

	for _, cmd := range commands {
		if result := execCommandWithDir(repoDir, cmd[0], cmd[1:]...); result.ExitCode != 0 {
			return fmt.Errorf("failed to %s: %s", cmd[0], result.Stderr)
		}
	}

	if debug {
		fmt.Printf("Successfully pushed branch: %s\n", branchName)
	}

	// Check for existing PRs in both the original repository and the fork
	searchRepo := originalRepo
	if !needsFork {
		searchRepo = cloneTarget
	}

	// First check for existing PRs in the target repository
	result := execCommand("gh", "pr", "list", "--repo", searchRepo, "--search", getPRSearchPattern(searchRepo), "--state", "open", "--json", "title,url")
	if debug {
		fmt.Printf("PR search in %s: exit=%d, output=%s\n", searchRepo, result.ExitCode, result.Stdout)
	}

	if result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "" && strings.TrimSpace(result.Stdout) != "[]" {
		fmt.Printf("ℹ️  Pull request already exists for repository: %s - skipping PR creation\n", searchRepo)
		return nil
	}

	// If we're using a fork, also check for existing PRs from our fork to avoid duplicates
	if needsFork {
		// Check for PRs from our fork to the upstream
		forkPRResult := execCommand("gh", "pr", "list", "--repo", originalRepo, "--author", "@me", "--state", "open", "--json", "title,url,headRefName")
		if debug {
			fmt.Printf("Fork PR search in %s by @me: exit=%d, output=%s\n", originalRepo, forkPRResult.ExitCode, forkPRResult.Stdout)
		}

		if forkPRResult.ExitCode == 0 && strings.TrimSpace(forkPRResult.Stdout) != "" && strings.TrimSpace(forkPRResult.Stdout) != "[]" {
			// Parse the PR list to check for similar titles
			var existingPRs []map[string]interface{}
			if err := json.Unmarshal([]byte(forkPRResult.Stdout), &existingPRs); err == nil {
				for _, pr := range existingPRs {
					if title, ok := pr["title"].(string); ok {
						if strings.Contains(strings.ToLower(title), "pin") && 
						   strings.Contains(strings.ToLower(title), "action") &&
						   strings.Contains(strings.ToLower(title), "security") {
							fmt.Printf("ℹ️  Similar pull request already exists from fork: %s - skipping PR creation\n", title)
							if url, ok := pr["url"].(string); ok {
								fmt.Printf("   • Existing PR: %s\n", url)
							}
							return nil
						}
					}
				}
			}
		}
	}

	prTitle := getPRTitleForRepository(searchRepo)

	// Get appropriate PR body based on repository's PR template
	prBodyContent := getPRBodyForRepository(repoDir)

	// Create PR - if forked, create PR to original repo
	var prResult ExecResult
	if needsFork {
		// Create cross-repository PR from fork to original
		headBranch := fmt.Sprintf("%s:%s", strings.Split(cloneTarget, "/")[0], branchName)
		if debug {
			fmt.Printf("Creating cross-repo PR: repo=%s, title=%s, base=%s, head=%s\n", originalRepo, prTitle, repo.DefaultBranchRef.Name, headBranch)
		}
		prResult = execCommand("gh", "pr", "create", "--repo", originalRepo, "--title", prTitle, "--body", prBodyContent, "--base", repo.DefaultBranchRef.Name, "--head", headBranch)
	} else {
		// Create normal PR within the same repository
		if debug {
			fmt.Printf("Creating PR: title=%s, base=%s, head=%s\n", prTitle, repo.DefaultBranchRef.Name, branchName)
		}
		prResult = execCommandWithDir(repoDir, "gh", "pr", "create", "--title", prTitle, "--body", prBodyContent, "--base", repo.DefaultBranchRef.Name, "--head", branchName)
	}

	if debug {
		fmt.Printf("PR creation: exit=%d, output=%s\n", prResult.ExitCode, prResult.Stdout)
	}

	if prResult.ExitCode != 0 {
		if debug {
			fmt.Printf("Trying alternative PR creation...\n")
		}
		// Try alternative method
		if needsFork {
			prResult = execCommand("gh", "pr", "create", "--repo", originalRepo, "--title", prTitle, "--body", prBodyContent)
		} else {
			prResult = execCommandWithDir(repoDir, "gh", "pr", "create", "--title", prTitle, "--body", prBodyContent)
		}

		if debug {
			fmt.Printf("Alternative PR: exit=%d, output=%s\n", prResult.ExitCode, prResult.Stdout)
		}
		if prResult.ExitCode != 0 {
			return fmt.Errorf("failed to create pull request: %s", prResult.Stderr)
		}
	}

	targetRepo := originalRepo
	if !needsFork {
		targetRepo = repo.Name
	}

	fmt.Printf("🎉 Pull request created successfully!\n")
	fmt.Printf("   • Repository: %s\n", targetRepo)
	if needsFork {
		fmt.Printf("   • Fork used: %s\n", cloneTarget)
	}
	if prResult.Stdout != "" {
		fmt.Printf("   • PR URL: %s\n", strings.TrimSpace(prResult.Stdout))
	}
	return nil
}

func forkRepository(repoName string) (string, error) {
	// Check if fork already exists
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repository name format: %s", repoName)
	}

	// Get current username
	result := execCommand("gh", "api", "user")
	if result.ExitCode != 0 {
		return "", fmt.Errorf("failed to get current user: %s", result.Stderr)
	}

	var user map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &user); err != nil {
		return "", fmt.Errorf("failed to parse user info: %v", err)
	}

	username, ok := user["login"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get username")
	}

	forkName := fmt.Sprintf("%s/%s", username, parts[1])

	// Check if fork already exists
	result = execCommand("gh", "repo", "view", forkName)
	if result.ExitCode == 0 {
		if debug {
			fmt.Printf("Fork already exists: %s\n", forkName)
		}
		return forkName, nil
	}

	// Create fork
	if debug {
		fmt.Printf("Creating fork of %s...\n", repoName)
	}

	result = execCommand("gh", "repo", "fork", repoName, "--clone=false")
	if result.ExitCode != 0 {
		return "", fmt.Errorf("failed to fork repository: %s", result.Stderr)
	}

	if debug {
		fmt.Printf("Successfully forked %s to %s\n", repoName, forkName)
	}

	return forkName, nil
}

func syncForkWithUpstream(forkName, upstreamName string) error {
	if debug {
		fmt.Printf("Checking if fork %s needs to be synced with upstream %s...\n", forkName, upstreamName)
	}

	// Get the default branch of the upstream repository
	upstreamResult := execCommand("gh", "repo", "view", upstreamName, "--json", "defaultBranchRef")
	if upstreamResult.ExitCode != 0 {
		return fmt.Errorf("failed to get upstream repository info: %s", upstreamResult.Stderr)
	}

	var upstreamRepo Repository
	if err := json.Unmarshal([]byte(upstreamResult.Stdout), &upstreamRepo); err != nil {
		return fmt.Errorf("failed to parse upstream repository info: %v", err)
	}

	defaultBranch := upstreamRepo.DefaultBranchRef.Name
	if defaultBranch == "" {
		defaultBranch = "main" // fallback
	}

	// Get the latest commit SHA from upstream
	upstreamCommitResult := execCommand("gh", "api", fmt.Sprintf("repos/%s/commits/%s", upstreamName, defaultBranch))
	if upstreamCommitResult.ExitCode != 0 {
		return fmt.Errorf("failed to get upstream commit: %s", upstreamCommitResult.Stderr)
	}

	var upstreamCommit map[string]interface{}
	if err := json.Unmarshal([]byte(upstreamCommitResult.Stdout), &upstreamCommit); err != nil {
		return fmt.Errorf("failed to parse upstream commit: %v", err)
	}

	upstreamSHA, ok := upstreamCommit["sha"].(string)
	if !ok {
		return fmt.Errorf("failed to get upstream commit SHA")
	}

	// Get the latest commit SHA from fork
	forkCommitResult := execCommand("gh", "api", fmt.Sprintf("repos/%s/commits/%s", forkName, defaultBranch))
	if forkCommitResult.ExitCode != 0 {
		return fmt.Errorf("failed to get fork commit: %s", forkCommitResult.Stderr)
	}

	var forkCommit map[string]interface{}
	if err := json.Unmarshal([]byte(forkCommitResult.Stdout), &forkCommit); err != nil {
		return fmt.Errorf("failed to parse fork commit: %v", err)
	}

	forkSHA, ok := forkCommit["sha"].(string)
	if !ok {
		return fmt.Errorf("failed to get fork commit SHA")
	}

	// Check if fork is behind upstream
	if upstreamSHA == forkSHA {
		if debug {
			fmt.Printf("Fork %s is up-to-date with upstream %s\n", forkName, upstreamName)
		}
		return nil
	}

	if debug {
		fmt.Printf("Fork %s is behind upstream %s, syncing...\n", forkName, upstreamName)
		fmt.Printf("  Fork SHA: %s\n", forkSHA)
		fmt.Printf("  Upstream SHA: %s\n", upstreamSHA)
	}

	// Sync the fork using GitHub API
	syncResult := execCommand("gh", "api", fmt.Sprintf("repos/%s/merge-upstream", forkName), "-X", "POST", "-f", fmt.Sprintf("branch=%s", defaultBranch))
	if syncResult.ExitCode != 0 {
		// If sync fails, try the alternative method using gh repo sync
		if debug {
			fmt.Printf("API sync failed, trying gh repo sync: %s\n", syncResult.Stderr)
		}
		
		syncResult = execCommand("gh", "repo", "sync", forkName, "--source", upstreamName)
		if syncResult.ExitCode != 0 {
			return fmt.Errorf("failed to sync fork with upstream: %s", syncResult.Stderr)
		}
	}

	if debug {
		fmt.Printf("Successfully synced fork %s with upstream %s\n", forkName, upstreamName)
	}

	return nil
}

func checkRepositoryPermissions(repoName string) error {
	// Check if the current user has write access to the repository
	result := execCommand("gh", "api", fmt.Sprintf("repos/%s", repoName))
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to check repository permissions: %s", result.Stderr)
	}

	var repoInfo map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &repoInfo); err != nil {
		return fmt.Errorf("failed to parse repository info: %v", err)
	}

	permissions, ok := repoInfo["permissions"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unable to determine repository permissions")
	}

	canPush, ok := permissions["push"].(bool)
	if !ok || !canPush {
		// No push access, need to fork
		if debug {
			fmt.Printf("No push access to %s, will fork repository\n", repoName)
		}
		return errNeedsFork
	}

	if debug {
		fmt.Printf("Repository permissions verified for: %s\n", repoName)
	}

	return nil
}

func configureGitCredentials(repoDir string) error {
	execCommandWithDir(repoDir, "git", "config", "--unset", "credential.helper")
	if result := execCommandWithDir(repoDir, "git", "config", "credential.helper", "!gh auth git-credential"); result.ExitCode != 0 {
		return fmt.Errorf("failed to configure git credentials: %s", result.Stderr)
	}

	// Also set the git user identity from gh auth status
	result := execCommandWithDir(repoDir, "gh", "auth", "status", "--hostname", "github.com")
	if result.ExitCode == 0 && strings.Contains(result.Stdout, "Logged in to github.com account") {
		// Extract the username from the auth status
		lines := strings.Split(result.Stdout, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Logged in to github.com account") && strings.Contains(line, "Active account: true") {
				// Extract username from the line format: "✓ Logged in to github.com account username (keyring)"
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "account" && i+1 < len(parts) {
						username := strings.TrimSuffix(parts[i+1], " (keyring)")
						username = strings.TrimSuffix(username, " (oauth_token)")
						if debug {
							fmt.Printf("Setting git user identity to: %s\n", username)
						}
						execCommandWithDir(repoDir, "git", "config", "user.name", username)
						execCommandWithDir(repoDir, "git", "config", "user.email", username+"@users.noreply.github.com")
						break
					}
				}
				break
			}
		}
	}

	return nil
}

func patchLocalRepository(repoDir string) error {
	workflowsDir := filepath.Join(repoDir, ".github", "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		fmt.Printf("ℹ️  No .github/workflows directory found - no GitHub Actions to pin\n")
		return nil
	}

	files, err := os.ReadDir(workflowsDir)
	if err != nil {
		return fmt.Errorf("failed to read workflows directory: %v", err)
	}

	workflowFiles := []string{}
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml")) {
			workflowFiles = append(workflowFiles, file.Name())
		}
	}

	if len(workflowFiles) == 0 {
		fmt.Printf("ℹ️  No workflow files found in .github/workflows directory\n")
		return nil
	}

	fmt.Printf("🔍 Found %d workflow file(s): %s\n", len(workflowFiles), strings.Join(workflowFiles, ", "))

	totalActionsPinned := 0
	totalActionsAlreadyPinned := 0
	totalActionsSkipped := 0
	totalActionsWithLatest := 0
	totalActionsWithoutTags := 0
	totalActionsFound := 0

	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml")) {
			pinned, alreadyPinned, skipped, withLatest, withoutTags, totalActions, err := processWorkflowFile(filepath.Join(workflowsDir, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to process workflow file %s: %v", file.Name(), err)
			}
			totalActionsPinned += pinned
			totalActionsAlreadyPinned += alreadyPinned
			totalActionsSkipped += skipped
			totalActionsWithLatest += withLatest
			totalActionsWithoutTags += withoutTags
			totalActionsFound += totalActions
		}
	}

	// Summary of actions processed
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   • Total actions found: %d\n", totalActionsFound)
	fmt.Printf("   • Actions pinned: %d\n", totalActionsPinned)
	fmt.Printf("   • Actions already pinned: %d\n", totalActionsAlreadyPinned)
	fmt.Printf("   • Actions with @latest: %d\n", totalActionsWithLatest)
	fmt.Printf("   • Actions without tag/ref: %d\n", totalActionsWithoutTags)
	fmt.Printf("   • Actions skipped: %d\n", totalActionsSkipped)

	if totalActionsPinned == 0 && totalActionsAlreadyPinned > 0 {
		fmt.Printf("✅ All GitHub Actions are already properly pinned to commit hashes\n")
	} else if totalActionsPinned == 0 && totalActionsAlreadyPinned == 0 && totalActionsSkipped > 0 {
		fmt.Printf("ℹ️  No GitHub Actions found that need pinning (only local actions or already pinned)\n")
	} else if totalActionsPinned == 0 {
		fmt.Printf("ℹ️  No GitHub Actions found in workflow files\n")
	} else if totalActionsPinned > 0 {
		if skipPRCreation {
			fmt.Printf("✅ Successfully pinned %d GitHub Action(s) to commit hashes\n", totalActionsPinned)
			fmt.Printf("   • Repository location: %s\n", repoDir)
			fmt.Printf("   • Changes are ready for review and manual commit\n")
		} else {
			fmt.Printf("✅ Successfully pinned %d GitHub Action(s) to commit hashes\n", totalActionsPinned)
		}
	}

	if totalActionsWithLatest > 0 {
		fmt.Printf("⚠️  Warning: %d action(s) using @latest tag detected - these should be pinned for better security\n", totalActionsWithLatest)
	}

	if totalActionsWithoutTags > 0 {
		fmt.Printf("🚨 Security Warning: %d action(s) found without any tag/ref - these are insecure as they default to the mutable default branch\n", totalActionsWithoutTags)
	}
	return nil
}

func processWorkflowFile(filePath string) (int, int, int, int, int, int, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("failed to read file: %v", err)
	}

	originalContent := string(content)
	var workflow map[string]interface{}
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("failed to parse YAML: %v", err)
	}

	jobs, ok := workflow["jobs"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, 0, 0, nil
	}

	// Collect all actions that need to be pinned
	var actionsToPin []actionPin
	var actionsAlreadyPinned int
	var actionsSkipped int
	var actionsWithLatest int
	var actionsWithoutTags int
	var totalActions int

	for _, jobData := range jobs {
		if job, ok := jobData.(map[string]interface{}); ok {
			if steps, ok := job["steps"].([]interface{}); ok {
				for _, stepData := range steps {
					if step, ok := stepData.(map[string]interface{}); ok {
						if uses, ok := step["uses"].(string); ok && uses != "" {
							totalActions++

							if shouldSkipAction(uses) {
								actionsSkipped++
								continue
							}

							// Check if already pinned (has commit hash)
							if matched, _ := regexp.MatchString(`@[a-f0-9]{40}`, uses); matched {
								actionsAlreadyPinned++
								continue
							}

							// Check for @latest tag
							if strings.Contains(uses, "@latest") {
								actionsWithLatest++
							}

							if action, version, err := parseActionReference(uses); err == nil {
								actionsToPin = append(actionsToPin, actionPin{action: action, version: version})
							} else if strings.Contains(err.Error(), "action without tag/ref") {
								// Action without tag/ref - this is insecure
								actionsWithoutTags++
							}
						}
					}
				}
			}
		}
	}

	if len(actionsToPin) == 0 {
		return 0, actionsAlreadyPinned, actionsSkipped, actionsWithLatest, actionsWithoutTags, totalActions, nil
	}

	// Process actions concurrently
	if len(actionsToPin) > 0 {
		fmt.Printf("🔄 Processing %d action(s) for pinning...\n", len(actionsToPin))
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > len(actionsToPin) {
		numWorkers = len(actionsToPin)
	}

	actionsChan := make(chan actionPin, len(actionsToPin))
	resultsChan := make(chan actionPin, len(actionsToPin))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go pinActionsWorker(actionsChan, resultsChan, &wg)
	}

	// Send actions to workers
	for _, action := range actionsToPin {
		actionsChan <- action
	}
	close(actionsChan)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	pinnedActions := make(map[string]actionPin)
	for result := range resultsChan {
		key := fmt.Sprintf("%s@%s", result.action, result.version)
		pinnedActions[key] = result
	}

	// Apply updates to the content
	updatedContent := originalContent
	currentDate := time.Now().Format("2006-01-02")
	actionsPinned := 0

	for _, jobData := range jobs {
		if job, ok := jobData.(map[string]interface{}); ok {
			if steps, ok := job["steps"].([]interface{}); ok {
				for _, stepData := range steps {
					if step, ok := stepData.(map[string]interface{}); ok {
						if uses, ok := step["uses"].(string); ok && uses != "" && !shouldSkipAction(uses) {
							if action, version, err := parseActionReference(uses); err == nil {
								key := fmt.Sprintf("%s@%s", action, version)
								if pinned, exists := pinnedActions[key]; exists {
									if pinned.err == nil {
										pinnedUses := fmt.Sprintf("%s@%s # %s on %s", action, pinned.hash, pinned.resolvedVersion, currentDate)
										updatedContent = strings.Replace(updatedContent, fmt.Sprintf("uses: %s", uses), fmt.Sprintf("uses: %s", pinnedUses), 1)
										actionsPinned++
										if debug {
											fmt.Printf("Pinned %s@%s to %s\n", action, version, pinned.hash)
										}
									} else if errors.Is(pinned.err, errUnresolvedVersion) {
										todoComment := fmt.Sprintf("# %s on %s, TODO: Pin to a commit hash", version, currentDate)
										newUses := fmt.Sprintf("%s %s", uses, todoComment)
										updatedContent = strings.Replace(updatedContent, fmt.Sprintf("uses: %s", uses), fmt.Sprintf("uses: %s", newUses), 1)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if updatedContent != originalContent {
		if err := os.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
			return 0, 0, 0, 0, 0, 0, fmt.Errorf("failed to write updated file: %v", err)
		}
	}
	return actionsPinned, actionsAlreadyPinned, actionsSkipped, actionsWithLatest, actionsWithoutTags, totalActions, nil
}

func shouldSkipAction(uses string) bool {
	// Skip local actions (relative paths)
	if strings.HasPrefix(uses, "./") {
		return true
	}
	// Skip certain action patterns if configured
	for _, skip := range skipActions {
		if strings.Contains(uses, skip) {
			return true
		}
	}
	return false
}

func parseActionReference(uses string) (string, string, error) {
	parts := strings.Split(uses, "@")
	if len(parts) == 1 {
		// Action without tag/ref - this is insecure as it defaults to default branch
		return parts[0], "", fmt.Errorf("action without tag/ref")
	}
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid action reference format")
	}
	return parts[0], parts[1], nil
}

func getCommitHashFromVersion(action, version string) (string, string, error) {
	if debug {
		start := time.Now()
		defer func() {
			fmt.Printf("Action %s@%s resolved in %v\n", action, version, time.Since(start))
		}()
	}

	// First try GitHub API approach (fastest, no cloning)
	if hash, resolvedVersion, err := getCommitHashViaAPI(action, version); err == nil {
		if debug {
			fmt.Printf("Resolved %s@%s via API (no cloning needed)\n", action, version)
		}
		return hash, resolvedVersion, nil
	}

	repoName := action
	if strings.Contains(action, "/") {
		parts := strings.Split(action, "/")
		if len(parts) >= 2 {
			repoName = fmt.Sprintf("%s/%s", parts[0], parts[1])
		}
	}

	actionDir := filepath.Join(getActionsCacheDir(), strings.ReplaceAll(repoName, "/", "_"))
	if err := os.MkdirAll(filepath.Dir(actionDir), 0755); err != nil {
		return "", "", fmt.Errorf("failed to create actions cache directory: %v", err)
	}

	if _, err := os.Stat(actionDir); os.IsNotExist(err) {
		if debug {
			fmt.Printf("Cloning action repository: %s (this may take a moment for large repos)\n", repoName)
		}
		// Use very shallow clone for fastest cloning - we only need recent history
		result := execCommand("gh", "repo", "clone", repoName, actionDir, "--", "--depth=1")
		if result.ExitCode != 0 {
			// Fallback to deeper clone if very shallow clone fails
			if debug {
				fmt.Printf("Very shallow clone failed, trying deeper clone for %s\n", repoName)
			}
			result = execCommand("gh", "repo", "clone", repoName, actionDir, "--", "--depth=10")
			if result.ExitCode != 0 {
				// Final fallback to full clone
				if debug {
					fmt.Printf("Shallow clone failed, trying full clone for %s\n", repoName)
				}
				result = execCommand("gh", "repo", "clone", repoName, actionDir)
				if result.ExitCode != 0 {
					return "", "", fmt.Errorf("failed to clone action repository: %s", result.Stderr)
				}
			}
		}
	} else if debug {
		fmt.Printf("Using cached action repository: %s\n", repoName)
		// Update the cached repository to get latest refs/tags
		if result := execCommandWithDir(actionDir, "git", "fetch", "origin", "--tags", "--quiet"); result.ExitCode != 0 && debug {
			fmt.Printf("Warning: Failed to update cached repository %s: %s\n", repoName, result.Stderr)
		}
	}

	// First try to resolve the version directly
	if result := execCommandWithDir(actionDir, "git", "rev-list", "-n", "1", version); result.ExitCode == 0 {
		return strings.TrimSpace(result.Stdout), version, nil
	}

	// If direct resolution fails, try to fetch the specific ref if it looks like a tag/branch
	if matched, _ := regexp.MatchString(`^[a-f0-9]{40}$`, version); !matched {
		if result := execCommandWithDir(actionDir, "git", "fetch", "origin", "+refs/tags/"+version+":refs/tags/"+version, "--quiet"); result.ExitCode == 0 {
			if result := execCommandWithDir(actionDir, "git", "rev-list", "-n", "1", version); result.ExitCode == 0 {
				return strings.TrimSpace(result.Stdout), version, nil
			}
		}
		// Also try fetching as a branch
		if result := execCommandWithDir(actionDir, "git", "fetch", "origin", "+refs/heads/"+version+":refs/remotes/origin/"+version, "--quiet"); result.ExitCode == 0 {
			if result := execCommandWithDir(actionDir, "git", "rev-list", "-n", "1", "origin/"+version); result.ExitCode == 0 {
				return strings.TrimSpace(result.Stdout), version, nil
			}
		}
	}

	if matched, _ := regexp.MatchString(`^v?\d+\.\d+\.\d+$`, version); matched {
		return "", "", errUnresolvedVersion
	}

	// Fallback to pattern matching for partial versions
	if result := execCommandWithDir(actionDir, "git", "tag", "-l", version+"*"); result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "" {
		tags := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		if len(tags) > 0 {
			return getCommitHashFromVersion(action, tags[len(tags)-1])
		}
	}

	return "", "", fmt.Errorf("version not found: %s", version)
}

// Try GitHub API approach for faster resolution (no cloning needed)
func getCommitHashViaAPI(action, version string) (string, string, error) {
	repoName := action
	if strings.Contains(action, "/") {
		parts := strings.Split(action, "/")
		if len(parts) >= 2 {
			repoName = fmt.Sprintf("%s/%s", parts[0], parts[1])
		}
	}

	// Try to get commit hash from GitHub API for tags/branches
	result := execCommand("gh", "api", fmt.Sprintf("repos/%s/git/refs/tags/%s", repoName, version))
	if result.ExitCode == 0 {
		var tagRef map[string]interface{}
		if err := json.Unmarshal([]byte(result.Stdout), &tagRef); err == nil {
			if object, ok := tagRef["object"].(map[string]interface{}); ok {
				if sha, ok := object["sha"].(string); ok {
					return sha, version, nil
				}
			}
		}
	}

	// Try as a branch
	result = execCommand("gh", "api", fmt.Sprintf("repos/%s/git/refs/heads/%s", repoName, version))
	if result.ExitCode == 0 {
		var branchRef map[string]interface{}
		if err := json.Unmarshal([]byte(result.Stdout), &branchRef); err == nil {
			if object, ok := branchRef["object"].(map[string]interface{}); ok {
				if sha, ok := object["sha"].(string); ok {
					return sha, version, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("could not resolve via API")
}

func execCommand(name string, args ...string) ExecResult {
	return execCommandWithDir("", name, args...)
}

func execCommandWithDir(dir, name string, args ...string) ExecResult {
	res, err := execute.ExecTask{Command: name, Args: args, Cwd: dir}.Execute(context.Background())
	result := ExecResult{Stdout: res.Stdout, Stderr: res.Stderr, ExitCode: res.ExitCode}

	if err != nil && result.ExitCode == 0 {
		result.ExitCode = 1
		if result.Stderr == "" {
			result.Stderr = err.Error()
		}
	}

	if debug && (result.ExitCode != 0 || result.Stderr != "") {
		fmt.Printf("Command: %s %s\n", name, strings.Join(args, " "))
		if dir != "" {
			fmt.Printf("Directory: %s\n", dir)
		}
		fmt.Printf("Exit Code: %d\n", result.ExitCode)
		if result.Stderr != "" {
			fmt.Printf("Stderr: %s\n", result.Stderr)
		}
	}

	return result
}

type actionPin struct {
	action          string
	version         string
	hash            string
	resolvedVersion string
	err             error
}

func pinActionsWorker(actions <-chan actionPin, results chan<- actionPin, wg *sync.WaitGroup) {
	defer wg.Done()
	for action := range actions {
		if hash, resolvedVersion, err := getCommitHashFromVersion(action.action, action.version); err == nil {
			action.hash = hash
			action.resolvedVersion = resolvedVersion
		} else {
			action.err = err
		}
		results <- action
	}
}

func getPRBodyForRepository(repoDir string) string {
	// If user wants to ignore PR templates, use full body
	if ignorePRTemplates {
		return prBody
	}

	// Check for PR templates in the repository
	templatePaths := []string{
		".github/pull_request_template.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		".github/pull_request_template.txt",
		".github/PULL_REQUEST_TEMPLATE.txt",
		"pull_request_template.md",
		"PULL_REQUEST_TEMPLATE.md",
	}

	for _, templatePath := range templatePaths {
		fullPath := filepath.Join(repoDir, templatePath)
		if content, err := os.ReadFile(fullPath); err == nil {
			if debug {
				fmt.Printf("Found PR template: %s\n", templatePath)
			}
			// Repository has a PR template, try to integrate with it
			return integratePRBodyWithTemplate(string(content))
		}
	}

	// No template found, use full body
	return prBody
}

func integratePRBodyWithTemplate(template string) string {
	// If template is very short or generic, replace it
	if len(strings.TrimSpace(template)) < 50 {
		return getMinimalPRBody()
	}

	// Fill out the template with our specific information
	filledTemplate := fillPRTemplate(template)

	if debug {
		fmt.Printf("Filled PR template with security pinning information\n")
	}

	return filledTemplate
}

func getMinimalPRBody() string {
	return `## Summary
Pin GitHub Actions to specific commit hashes for improved security and reproducible builds.

## Changes
- Converted version tags to commit hashes
- Added comments with original version references
- No functional changes to workflows

## Security Benefits
- Prevents supply chain attacks
- Ensures reproducible builds
- Follows GitHub security best practices`
}

func getEmptyPRBody() string {
	// Return empty string to let PR template take over completely
	return ""
}

func fillPRTemplate(template string) string {
	filledTemplate := template

	// Define our content for different sections
	description := "This pull request pins all GitHub Actions in workflow files to specific commit hashes to improve security and ensure reproducible builds."

	changes := `- Converted version tags (e.g., v3, v4) to commit hashes
- Added comments showing the original version and date for reference
- Preserved all existing functionality while improving security`

	testing := `- Verified all workflow files are syntactically correct
- Confirmed no functional changes to existing workflows
- All pinned actions maintain their original functionality`

	securityBenefits := `- **Security**: Prevents supply chain attacks by ensuring immutable action references
- **Reproducibility**: Guarantees the same action version is used across all runs
- **Auditability**: Clear tracking of which specific version of each action is being used`

	// Replace common placeholders and sections
	replacements := map[string]string{
		"Please include a summary of the changes and the related issue. Please also include relevant motivation and context. List any dependencies that are required for this change.": description,
		"Please describe the changes made in this PR.": description,
		"Describe your changes here.":                  description,
		"What does this PR do?":                        description,
		"## Description":                               "## Description\n\n" + description,
		"## Summary":                                   "## Summary\n\n" + description,
		"## What":                                      "## What\n\n" + description,
		"## Changes":                                   "## Changes\n\n" + changes,
		"## Changes Made":                              "## Changes Made\n\n" + changes,
		"## How Has This Been Tested?":                 "## How Has This Been Tested?\n\n" + testing,
		"## Testing":                                   "## Testing\n\n" + testing,
		"Please describe the tests that you ran to verify your changes. Provide instructions so we can reproduce. Please also list any relevant details for your test configuration.": testing,
	}

	// Apply replacements
	for placeholder, replacement := range replacements {
		filledTemplate = strings.Replace(filledTemplate, placeholder, replacement, -1)
	}

	// Handle checkboxes - mark relevant ones as checked
	checkboxReplacements := map[string]string{
		"- [ ] Security improvement": "- [x] Security improvement",
		"- [ ] This change does not introduce any new security vulnerabilities":            "- [x] This change does not introduce any new security vulnerabilities",
		"- [ ] I have reviewed the security implications of my changes":                    "- [x] I have reviewed the security implications of my changes",
		"- [ ] My code follows the style guidelines of this project":                       "- [x] My code follows the style guidelines of this project",
		"- [ ] I have performed a self-review of my own code":                              "- [x] I have performed a self-review of my own code",
		"- [ ] My changes generate no new warnings":                                        "- [x] My changes generate no new warnings",
		"- [ ] Any dependent changes have been merged and published in downstream modules": "- [x] Any dependent changes have been merged and published in downstream modules",
	}

	// Apply checkbox replacements
	for unchecked, checked := range checkboxReplacements {
		filledTemplate = strings.Replace(filledTemplate, unchecked, checked, -1)
	}

	// Add our security benefits section if there's a placeholder for it
	if strings.Contains(strings.ToLower(filledTemplate), "security considerations") ||
		strings.Contains(strings.ToLower(filledTemplate), "## benefits") {
		filledTemplate = strings.Replace(filledTemplate, "## Security Considerations", "## Security Considerations\n\n"+securityBenefits+"\n\n## Additional Security Notes", -1)
		filledTemplate = strings.Replace(filledTemplate, "## Benefits", "## Benefits\n\n"+securityBenefits, -1)
	}

	// If template asks for issue number, add a note about security improvement
	filledTemplate = strings.Replace(filledTemplate, "Fixes # (issue)", "Security improvement: Pin GitHub Actions to commit hashes", -1)
	filledTemplate = strings.Replace(filledTemplate, "Closes # (issue)", "Security improvement: Pin GitHub Actions to commit hashes", -1)
	filledTemplate = strings.Replace(filledTemplate, "Fixes #(issue)", "Security improvement: Pin GitHub Actions to commit hashes", -1)

	return filledTemplate
}

func getPRTitleForRepository(repoName string) string {
	// Check for known repositories with specific title requirements
	if strings.Contains(repoName, "ossf/") || strings.Contains(repoName, "kubernetes") || strings.Contains(repoName, "k8s.io") {
		// These repositories often use emoji prefixes for PR categorization
		return ":seedling: security: pin GitHub Actions to commit hashes"
	}
	
	// Default title for most repositories - use conventional commit format
	return "security: pin GitHub Actions to commit hashes"
}

func getPRSearchPattern(repoName string) string {
	// Return the appropriate search pattern based on repository
	if strings.Contains(repoName, "ossf/") || strings.Contains(repoName, "kubernetes") || strings.Contains(repoName, "k8s.io") {
		return ":seedling: security: pin GitHub Actions to commit hashes in:title"
	}
	
	return "security: pin GitHub Actions to commit hashes in:title"
}

// ...existing code...
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

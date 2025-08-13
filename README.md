# GHA Pinner

A command-line tool for pinning GitHub Actions to specific commit hashes to improve security and ensure reproducible builds.

## Features

- **Action Pinning**: Converts version tags (e.g., `v3`, `v4`) to commit hashes
- **Batch File Processing**: Process multiple repositories from a file list
- **Organization Processing**: Process entire organizations or individual repositories
- **Local Repository Support**: Pin actions in local repositories
- **Fork Synchronization**: Automatically syncs forks with upstream before processing
- **Automated PR Creation**: Creates pull requests with pinned actions
- **No-PR Mode**: Skip PR creation and only fix repositories locally for review
- **Custom Output Directory**: Specify where to save repositories in no-PR mode
- **Enhanced Duplicate Prevention**: Detects existing PRs to avoid duplicates
- **PR Template Support**: Automatically detects and fills out repository PR templates
- **Fork Support**: Automatically forks repositories when write access is not available
- **Performance Optimized**: Uses GitHub API for faster resolution and caches action repositories
- **Security Focused**: Prevents supply chain attacks through immutable references
- **Comment Preservation**: Shows original version and date for auditing
- **Account Switching**: Switch between different GitHub accounts
- **Concurrent Processing**: Processes multiple actions simultaneously for better performance
- **Flexible URL Parsing**: Supports various GitHub URL formats (HTTPS, SSH, owner/repo)

## Installation

### Prerequisites

- Go 1.21 or later
- GitHub CLI (`gh`) installed and authenticated
- Git installed and configured

### Install via go install (Recommended)

```bash
go install github.com/harekrishnarai/gha-pinner/cmd/gha-pinner@latest
```

This will install the `gha-pinner` binary to your `$GOPATH/bin` directory (typically `~/go/bin`). Make sure this directory is in your `PATH`.

### Build from Source

```bash
git clone https://github.com/harekrishnarai/gha-pinner
cd gha-pinner
go build -o gha-pinner ./cmd/gha-pinner
```

## Usage

### Commands

```bash
# Pin actions in a local repository
gha-pinner local-repository <path> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]

# Pin actions in a remote repository
gha-pinner repository <repo-name> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]

# Pin actions in all repositories of an organization
gha-pinner organization <org-name> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]

# Process multiple repositories from a file
gha-pinner file <path-to-repos-file> [--debug] [--ignore-templates] [--no-pr] [--output <dir>]

# Resolve a specific action version to commit hash
gha-pinner action <action-name> <version> [--debug]

# Switch between GitHub accounts
gha-pinner switch-account <username> [--debug]
```

### Options

- `--debug`: Enable debug output with timing information
- `--ignore-templates`: Ignore PR templates and use full PR body instead of filling templates
- `--no-pr`: Skip PR creation, only fix repositories locally for manual review
- `--output <dir>`: Custom output directory for repositories (only with --no-pr)

### Examples

```bash
# Process local repository
gha-pinner local-repository ./my-repo

# Process remote repository
gha-pinner repository owner/repo-name

# Process entire organization
gha-pinner organization my-org

# Process multiple repositories from a file
gha-pinner file repos.txt

# Process repositories but skip PR creation (for manual review)
gha-pinner file repos.txt --no-pr

# Process with custom output directory
gha-pinner file repos.txt --no-pr --output ./fixed-repos

# Resolve specific action version
gha-pinner action actions/checkout v3

# Switch GitHub account
gha-pinner switch-account myusername

# Enable debug output
gha-pinner repository owner/repo-name --debug

# Ignore PR templates and use full description
gha-pinner repository owner/repo-name --ignore-templates
```

### Repository File Format

When using the `file` command, create a text file with one repository URL per line:

```text
# Security-focused repositories
https://github.com/slsa-framework/slsa
https://github.com/ossf/scorecard
https://github.com/sigstore/cosign
owner/repository-name
```

The tool supports various URL formats:
- `https://github.com/owner/repo`
- `https://github.com/owner/repo.git`
- `git@github.com:owner/repo.git`
- `owner/repo` (simple format)

Comments (lines starting with `#`) and empty lines are ignored.

## How It Works

### Action Pinning Process

1. **Discovery**: Scans `.github/workflows/*.yml` files for GitHub Actions
2. **Resolution**: Resolves version tags to specific commit hashes
3. **Replacement**: Replaces version tags with commit hashes while preserving original version in comments
4. **Validation**: Ensures all changes maintain workflow functionality

### Example Transformation

**Before:**
```yaml
- name: Checkout code
  uses: actions/checkout@v3
```

**After:**
```yaml
- name: Checkout code
  uses: actions/checkout@f43a0e5ff2bd294095638e18286ca9a3d1956744 # v3 on 2025-06-27
```

### Repository Processing

1. **Permission Check**: Verifies write access, forks repository if needed
2. **Fork Synchronization**: If using a fork, syncs with upstream to ensure latest code
3. **Clone**: Clones the target repository (or fork) to a temporary directory
4. **Patch**: Processes all workflow files to pin actions concurrently
5. **Branch**: Creates a new branch with timestamp for uniqueness
6. **Commit**: Commits changes with natural-sounding commit message
7. **Push**: Pushes branch to origin (or fork)
8. **Duplicate Detection**: Checks for existing PRs to avoid duplicates
9. **PR Template Detection**: Detects and fills out repository PR templates
10. **PR**: Creates pull request with filled template or custom description

### Batch Processing Workflow

When using the `file` command for batch processing:

1. **File Parsing**: Reads repository URLs from the file, skipping comments and empty lines
2. **URL Normalization**: Converts various GitHub URL formats to standard `owner/repo` format
3. **Sequential Processing**: Processes repositories one by one with progress reporting
4. **Error Isolation**: Continues processing other repositories if one fails
5. **Summary Report**: Provides success/failure statistics at the end

### No-PR Mode

When using `--no-pr` flag:

1. **Local Processing**: Performs all repository operations without creating PRs
2. **Change Preview**: Shows a diff of all changes made to workflow files
3. **Repository Preservation**: Keeps repositories in temp directory (or custom output) for manual review
4. **Manual Instructions**: Provides commands for manual commit and push operations

This mode is perfect for:
- Testing changes before creating PRs
- Reviewing multiple repositories at once
- Custom workflow scenarios
- Compliance requirements

### PR Template Support

The tool automatically detects PR templates in repositories and intelligently fills them out:

- **Template Detection**: Searches for common PR template files (`.github/pull_request_template.md`, etc.)
- **Smart Filling**: Fills out description, testing, and security sections with relevant information
- **Checkbox Handling**: Automatically checks relevant boxes (security improvement, code review, etc.)
- **Professional Output**: Creates PRs that look manually crafted rather than automated

Use `--ignore-templates` to bypass template detection and use the full custom PR body instead.

## Security Benefits

### Supply Chain Attack Prevention

- **Immutable References**: Commit hashes cannot be changed, preventing malicious updates
- **Reproducible Builds**: Ensures the exact same action version across all runs
- **Audit Trail**: Clear tracking of which specific version is being used with dates

### Best Practices Compliance

- Follows GitHub's recommended security practices
- Maintains backward compatibility
- Preserves original version information for reference
- Natural commit messages and PR descriptions to avoid detection as automated tools

## Performance Features

### Optimization Strategies

- **GitHub API First**: Attempts to resolve versions via API before cloning repositories
- **Action Repository Caching**: Caches cloned action repositories for faster subsequent runs
- **Shallow Clones**: Uses minimal git history for faster cloning
- **Concurrent Processing**: Processes multiple actions within workflows simultaneously
- **Persistent Cache**: Maintains cache between runs for repeated operations

### Debug Output

Enable `--debug` flag to see detailed timing information:
- Individual action resolution times
- API vs clone resolution methods
- Cache hits and misses
- Total execution time

### Configuration

### Command Line Options

- `--debug`: Enable detailed debug output with timing information
- `--ignore-templates`: Skip PR template detection and use full custom PR body
- `--no-pr`: Skip PR creation and only fix repositories locally for manual review
- `--output <dir>`: Custom output directory for repositories (only effective with --no-pr)

### Environment Variables

- `DEBUG`: Enable debug output (alternative to `--debug` flag)

### GitHub CLI Requirements

The tool requires GitHub CLI to be installed and authenticated with proper scopes:

```bash
# Install GitHub CLI
# See: https://cli.github.com/

# Authenticate with GitHub (ensure you have repo and workflow scopes)
gh auth login

# Check authentication status
gh auth status

# Switch between accounts if needed
gha-pinner switch-account username
```

**Important**: Make sure your GitHub token has the following scopes:
- `repo`: Full control of repositories (required for forking and creating PRs)
- `workflow`: Update GitHub Action workflows
- `read:org`: Read organization and team membership (for organization processing)

### Repository File Configuration

Create a `repos.txt` file with repositories you want to process:

```text
# SLSA and Supply Chain Security
https://github.com/slsa-framework/slsa
https://github.com/ossf/scorecard
https://github.com/in-toto/in-toto

# Sigstore Ecosystem
https://github.com/sigstore/cosign
https://github.com/sigstore/fulcio
https://github.com/sigstore/rekor

# Vulnerability Scanning
https://github.com/anchore/grype
https://github.com/aquasecurity/trivy
https://github.com/google/osv-scanner
```

The included `repos.txt` contains 50 curated security-focused repositories for testing.

## Advanced Usage

### Batch Processing Workflow

For processing multiple repositories efficiently:

```bash
# 1. Review the repository list
cat repos.txt

# 2. Test with no-PR mode first
gha-pinner file repos.txt --no-pr --debug

# 3. Review changes in preserved repositories
ls -la /tmp/repos/

# 4. When satisfied, run with PR creation
gha-pinner file repos.txt --debug
```

### Custom Workflow Examples

```bash
# Process with custom output for compliance review
gha-pinner file security-repos.txt --no-pr --output ./compliance-review

# Process organization with no-PR for testing
gha-pinner organization my-org --no-pr --debug

# Process single repository without PR for manual testing
gha-pinner repository critical-app/main --no-pr

# Process and ignore PR templates for consistent formatting
gha-pinner file repos.txt --ignore-templates
```

### Integration with CI/CD

```bash
# In CI environment, use no-PR mode and commit to branch
gha-pinner file repos.txt --no-pr --output ./pinned-repos
cd pinned-repos/my-repo
git add .
git commit -m "security: pin GitHub Actions to commit hashes"
git push origin security-pinning-branch
```

### Monitoring and Reporting

```bash
# Generate detailed logs for security auditing
gha-pinner file repos.txt --debug > security-pinning-report.log 2>&1

# Process with progress tracking
gha-pinner file repos.txt --debug | tee security-improvements.log
```

### Cache Location

Action repositories are cached in:
- **Linux/macOS**: `/tmp/gha-pinner-cache/actions/`
- **Windows**: `%TEMP%\gha-pinner-cache\actions\`

## Development

### Project Structure

```
gha-pinner/
├── cmd/
│   └── gha-pinner/
│       ├── main.go          # Main application logic
│       ├── main_test.go     # Unit tests
│       └── pr-body.md       # Template PR body
├── go.mod                   # Go module definition
├── go.sum                   # Go dependencies
└── README.md               # This file
```

### Architecture

The application follows a modular design with clear separation of concerns:

- **Command Processing**: Handles CLI argument parsing and command routing
- **Repository Operations**: Manages git operations and GitHub API interactions
- **Workflow Processing**: Parses and modifies YAML workflow files
- **Action Resolution**: Resolves version tags to commit hashes
- **Error Handling**: Comprehensive error handling with cleanup

### Running Tests

```bash
go test ./cmd/gha-pinner -v
```

### Building

```bash
# Build for current platform
go build -o gha-pinner ./cmd/gha-pinner

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o gha-pinner-linux ./cmd/gha-pinner
GOOS=windows GOARCH=amd64 go build -o gha-pinner-windows.exe ./cmd/gha-pinner
GOOS=darwin GOARCH=amd64 go build -o gha-pinner-macos ./cmd/gha-pinner
```

## Error Handling

The tool includes comprehensive error handling:

- **Graceful Degradation**: Continues processing other repositories on individual failures
- **Fork Synchronization**: Automatically syncs forks with upstream to prevent outdated PRs
- **Duplicate Prevention**: Detects existing PRs to avoid creating duplicates
- **Authentication Handling**: Provides clear error messages for token permission issues
- **Cleanup**: Removes temporary directories on exit (except in --no-pr mode)
- **Detailed Logging**: Debug mode shows full execution traces
- **Recovery**: Handles common failure scenarios (network issues, authentication, etc.)
- **Progress Reporting**: Shows real-time progress for batch operations

### Common Issues and Solutions

**403 Forbidden Errors**: 
- Ensure your GitHub token has proper scopes (`repo`, `workflow`)
- Re-authenticate with `gh auth login` if using environment tokens

**Fork Permission Issues**:
- Tool automatically handles forking when you don't have write access
- Syncs forks with upstream to ensure latest code

**Duplicate PRs**:
- Tool checks for existing PRs before creating new ones
- Enhanced detection includes PRs from forks to upstream repositories

## Limitations

- Requires GitHub CLI for authentication and API access
- Only processes public repositories or those accessible with current authentication
- Some semantic versions may not be resolvable to specific commits
- Rate limiting may apply for large organizations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues, questions, or contributions, please:

1. Check existing issues on GitHub
2. Create a new issue with detailed information
3. Include debug output when reporting problems

## Changelog

### v2.0.0 (Latest)
- **New**: Batch file processing with `file` command for multiple repositories
- **New**: Fork synchronization to prevent duplicate PRs and ensure latest code
- **New**: `--no-pr` flag for local processing without PR creation
- **New**: `--output` flag for custom output directories in no-PR mode
- **New**: Enhanced duplicate PR prevention with fork detection
- **New**: Support for various GitHub URL formats (HTTPS, SSH, owner/repo)
- **New**: Comprehensive progress reporting for batch operations
- **New**: Change preview in no-PR mode with diff output
- **Improved**: Better error handling and authentication guidance
- **Improved**: Enhanced repository preservation for manual review
- **Added**: Sample `repos.txt` with 50 security-focused repositories

### v1.0.0
- Initial release
- Support for local repository processing
- Support for remote repository processing
- Support for organization-wide processing
- Action version resolution
- Automated PR creation
- Comprehensive error handling

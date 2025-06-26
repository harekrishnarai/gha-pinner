# GHA Pinner

A command-line tool for pinning GitHub Actions to specific commit hashes to improve security and ensure reproducible builds.

## Features

- **Action Pinning**: Converts version tags (e.g., `v3`, `v4`) to commit hashes
- **Batch Processing**: Process entire organizations or individual repositories
- **Local Repository Support**: Pin actions in local repositories
- **Automated PR Creation**: Creates pull requests with pinned actions
- **PR Template Support**: Automatically detects and fills out repository PR templates
- **Fork Support**: Automatically forks repositories when write access is not available
- **Performance Optimized**: Uses GitHub API for faster resolution and caches action repositories
- **Security Focused**: Prevents supply chain attacks through immutable references
- **Comment Preservation**: Shows original version and date for auditing
- **Account Switching**: Switch between different GitHub accounts
- **Concurrent Processing**: Processes multiple actions simultaneously for better performance

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
gha-pinner local-repository <path> [--debug] [--ignore-templates]

# Pin actions in a remote repository
gha-pinner repository <repo-name> [--debug] [--ignore-templates]

# Pin actions in all repositories of an organization
gha-pinner organization <org-name> [--debug] [--ignore-templates]

# Resolve a specific action version to commit hash
gha-pinner action <action-name> <version> [--debug]

# Switch between GitHub accounts
gha-pinner switch-account <username> [--debug]
```

### Options

- `--debug`: Enable debug output with timing information
- `--ignore-templates`: Ignore PR templates and use full PR body instead of filling templates

### Examples

```bash
# Process local repository
gha-pinner local-repository ./my-repo

# Process remote repository
gha-pinner repository owner/repo-name

# Process entire organization
gha-pinner organization my-org

# Resolve specific action version
gha-pinner action actions/checkout v3

# Switch GitHub account
gha-pinner switch-account myusername

# Enable debug output
gha-pinner repository owner/repo-name --debug

# Ignore PR templates and use full description
gha-pinner repository owner/repo-name --ignore-templates
```

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

1. **Clone**: Clones the target repository to a temporary directory
2. **Permission Check**: Verifies write access, forks repository if needed
3. **Patch**: Processes all workflow files to pin actions concurrently
4. **Branch**: Creates a new branch with timestamp for uniqueness
5. **Commit**: Commits changes with natural-sounding commit message
6. **Push**: Pushes branch to origin (or fork)
7. **PR Template Detection**: Detects and fills out repository PR templates
8. **PR**: Creates pull request with filled template or custom description

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

## Configuration

### Command Line Options

- `--debug`: Enable detailed debug output with timing information
- `--ignore-templates`: Skip PR template detection and use full custom PR body

### Environment Variables

- `DEBUG`: Enable debug output (alternative to `--debug` flag)

### GitHub CLI Requirements

The tool requires GitHub CLI to be installed and authenticated:

```bash
# Install GitHub CLI
# See: https://cli.github.com/

# Authenticate with GitHub
gh auth login

# Switch between accounts if needed
gha-pinner switch-account username
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
- **Cleanup**: Removes temporary directories on exit
- **Detailed Logging**: Debug mode shows full execution traces
- **Recovery**: Handles common failure scenarios (network issues, authentication, etc.)

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

### v1.0.0
- Initial release
- Support for local repository processing
- Support for remote repository processing
- Support for organization-wide processing
- Action version resolution
- Automated PR creation
- Comprehensive error handling

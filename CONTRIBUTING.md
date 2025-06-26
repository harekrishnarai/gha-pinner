# Contributing to GHA Pinner

Thank you for your interest in contributing to GHA Pinner! This document provides guidelines and information to help you contribute effectively.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Community and Support](#community-and-support)

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

### Our Pledge

We are committed to making participation in our project a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Expected Behavior

- Use welcoming and inclusive language
- Be respectful of differing viewpoints and experiences
- Gracefully accept constructive criticism
- Focus on what is best for the community
- Show empathy towards other community members

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- [Go](https://golang.org/doc/install) (version 1.21 or later)
- [Git](https://git-scm.com/downloads)
- [GitHub CLI](https://cli.github.com/) (authenticated)
- A GitHub account

### Forking the Repository

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gha-pinner.git
   cd gha-pinner
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/harekrishnarai/gha-pinner.git
   ```

## Development Setup

1. **Install dependencies:**
   ```bash
   go mod download
   ```

2. **Build the project:**
   ```bash
   go build -o gha-pinner ./cmd/gha-pinner
   ```

3. **Run tests:**
   ```bash
   go test ./...
   ```

4. **Run with race detection:**
   ```bash
   go test -race ./...
   ```

## How to Contribute

### Reporting Bugs

- Check if the bug has already been reported in [Issues](https://github.com/harekrishnarai/gha-pinner/issues)
- If not, create a new issue using the bug report template
- Include as much detail as possible: steps to reproduce, expected vs actual behavior, environment details

### Suggesting Features

- Check if the feature has already been requested in [Issues](https://github.com/harekrishnarai/gha-pinner/issues)
- If not, create a new issue using the feature request template
- Clearly describe the feature and its benefits
- Consider discussing in [Discussions](https://github.com/harekrishnarai/gha-pinner/discussions) first for major features

### Contributing Code

1. **Find an issue to work on** or create one
2. **Comment on the issue** to let others know you're working on it
3. **Create a branch** for your work:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. **Make your changes** following the coding standards
5. **Write tests** for your changes
6. **Update documentation** if needed
7. **Test your changes** thoroughly
8. **Commit your changes** with clear commit messages
9. **Push your branch** and create a pull request

## Coding Standards

### Go Style Guidelines

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `go fmt` to format your code
- Use `go vet` to check for common errors
- Use `golangci-lint` for comprehensive linting

### Code Organization

- Keep functions small and focused
- Use descriptive variable and function names
- Add comments for complex logic
- Organize code into logical packages
- Follow the existing project structure

### Error Handling

- Always handle errors appropriately
- Use descriptive error messages
- Wrap errors with context when appropriate
- Don't ignore errors silently

### Example Code Style

```go
// Good
func processRepository(repoPath string) error {
    if repoPath == "" {
        return fmt.Errorf("repository path cannot be empty")
    }
    
    // Process repository logic here
    return nil
}

// Bad
func process(p string) error {
    // No validation, unclear naming
    return nil
}
```

## Testing

### Writing Tests

- Write tests for all new functionality
- Follow the table-driven test pattern when appropriate
- Use descriptive test names that explain what is being tested
- Include both positive and negative test cases
- Test edge cases and error conditions

### Test Organization

- Place tests in the same package as the code being tested
- Use `_test.go` suffix for test files
- Group related tests together
- Use setup and teardown functions when needed

### Example Test

```go
func TestProcessRepository(t *testing.T) {
    tests := []struct {
        name     string
        repoPath string
        wantErr  bool
    }{
        {
            name:     "valid repository path",
            repoPath: "/valid/path",
            wantErr:  false,
        },
        {
            name:     "empty repository path",
            repoPath: "",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := processRepository(tt.repoPath)
            if (err != nil) != tt.wantErr {
                t.Errorf("processRepository() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Run tests with race detection
go test -race ./...

# Run specific test
go test -run TestProcessRepository ./...
```

## Documentation

### Code Documentation

- Document all public functions, types, and constants
- Use Go doc conventions
- Include examples in documentation when helpful
- Keep documentation up to date with code changes

### README Updates

- Update the README.md if your changes affect:
  - Installation instructions
  - Usage examples
  - Feature descriptions
  - Configuration options

### Example Documentation

```go
// ProcessRepository processes a Git repository and pins GitHub Actions
// to specific commit hashes for improved security.
//
// The repoPath parameter must be a valid path to a Git repository.
// Returns an error if the repository cannot be processed.
//
// Example:
//   err := ProcessRepository("/path/to/repo")
//   if err != nil {
//       log.Fatal(err)
//   }
func ProcessRepository(repoPath string) error {
    // Implementation here
}
```

## Submitting Changes

### Commit Messages

Write clear, descriptive commit messages:

```
feat: add support for pinning composite actions

- Implement parsing for composite action references
- Add tests for composite action handling
- Update documentation with new feature

Fixes #123
```

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or changes
- `chore`: Build process or auxiliary tool changes

### Pull Request Process

1. **Ensure your branch is up to date:**
   ```bash
   git checkout main
   git pull upstream main
   git checkout your-branch
   git rebase main
   ```

2. **Create a pull request** using the provided template
3. **Fill out the PR template** completely
4. **Link related issues** using keywords like "Fixes #123"
5. **Request review** from maintainers
6. **Address feedback** promptly and professionally
7. **Keep the PR updated** with any requested changes

### PR Requirements

- [ ] All tests pass
- [ ] Code follows project standards
- [ ] Documentation is updated
- [ ] Commit messages are clear
- [ ] PR description is complete
- [ ] Related issues are linked

## Community and Support

### Getting Help

- **Documentation**: Check the [README](README.md) first
- **Discussions**: Use [GitHub Discussions](https://github.com/harekrishnarai/gha-pinner/discussions) for questions and ideas
- **Issues**: Use [GitHub Issues](https://github.com/harekrishnarai/gha-pinner/issues) for bug reports and feature requests

### Communication Guidelines

- Be respectful and professional
- Provide context and examples
- Search existing issues and discussions before posting
- Use clear, descriptive titles
- Follow up on your issues and PRs

### Recognition

Contributors are recognized in various ways:
- Listed in the project's contributors section
- Mentioned in release notes for significant contributions
- Invited to become maintainers for exceptional ongoing contributions

## License

By contributing to GHA Pinner, you agree that your contributions will be licensed under the same license as the project.

## Questions?

If you have questions about contributing, please:
1. Check this document first
2. Search existing issues and discussions
3. Create a new discussion or issue
4. Reach out to the maintainers

Thank you for contributing to GHA Pinner! 🚀
# Pin GitHub Actions to commit hashes

This pull request pins all GitHub Actions in workflow files to specific commit hashes to improve security and ensure reproducible builds.

## Changes Made

- Converted version tags (e.g., `v3`, `v4`) to commit hashes
- Added comments showing the original version for reference
- Preserved all existing functionality while improving security

## Benefits

- **Security**: Prevents supply chain attacks by ensuring immutable action references
- **Reproducibility**: Guarantees the same action version is used across all runs
- **Auditability**: Clear tracking of which specific version of each action is being used

## Review Notes

- All pinned actions maintain their original functionality
- Comments preserve the original version information for easy reference
- No workflow behavior changes are expected

This change follows GitHub's security best practices for action pinning.

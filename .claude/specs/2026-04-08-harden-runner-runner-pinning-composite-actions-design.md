# Design Spec: Harden Runner Injection, Runner Label Pinning & Composite Action Support

**Date:** 2026-04-08  
**Inspired by:** [actions-security-demo/poc-1#48](https://github.com/actions-security-demo/poc-1/pull/48)  
**Status:** Approved

---

## Problem

gha-pinner currently only pins GitHub Actions to commit SHAs. PR #48 from StepSecurity's tool reveals two additional supply-chain hardening controls that users want:

1. **Harden Runner injection** — adds `step-security/harden-runner` as the first step in every job to monitor/block outbound network calls at runtime.
2. **Runner label pinning** — replaces floating labels like `ubuntu-latest` with versioned equivalents like `ubuntu-24.04` to prevent unexpected runner environment changes.
3. **Composite action file support** — `.github/actions/**/*.yml` files contain `uses:` references that are currently not pinned.

All three are opt-in to avoid breaking existing workflows.

---

## New CLI Flags

| Flag | Type | Default | Purpose |
|---|---|---|---|
| `--inject-harden-runner` | bool | `false` | Inject `step-security/harden-runner` as first step in every job |
| `--egress-policy` | string | `"audit"` | Egress policy for injected harden-runner (`audit` or `block`) |
| `--pin-runners` | bool | `false` | Replace floating runner labels with versioned equivalents |
| `--runner-map` | `[]string` | nil | Custom label overrides, e.g. `ubuntu-latest=ubuntu-24.04` |

- `--egress-policy` is silently ignored if `--inject-harden-runner` is not set.
- `--runner-map` is silently ignored if `--pin-runners` is not set.
- `--egress-policy` only accepts `"audit"` or `"block"` — validated in `validateRuntimeConfig`.

---

## Architecture: `WorkflowPatcher` Struct

Replace the bare `processWorkflowFile` function with a method on a new struct:

```go
type WorkflowPatcher struct {
    injectHardenRunner bool
    egressPolicy       string   // "audit" or "block"
    pinRunners         bool
    runnerMap          map[string]string // parsed from --runner-map flag
}
```

`(p *WorkflowPatcher) patchFile(filePath string)` replaces `processWorkflowFile`. It runs three ordered passes on the raw file content string:

```
pass 1: pinActionsPass         — always runs (existing logic)
pass 2: injectHardenRunnerPass — runs only if p.injectHardenRunner == true
pass 3: pinRunnersPass         — runs only if p.pinRunners == true
```

Each pass receives the current content string and returns `(updatedContent string, changeCount int, err error)`. A single `os.WriteFile` at the end writes the final result — no intermediate writes.

A single `WorkflowPatcher` instance is constructed at command startup (from the resolved flag values) and reused across all files in the repository.

---

## Composite Action File Support

`patchLocalRepository` is extended to scan two paths:

1. `.github/workflows/*.yml` — existing behaviour
2. `.github/actions/**/*.yml` — new recursive scan

**YAML shape detection** inside `patchFile`:
- `workflow["jobs"]` present → jobs-based workflow file
- `workflow["runs"]` present → composite action file

For composite action files:
- `pinActionsPass` runs — walks `runs.steps` instead of `jobs.<id>.steps`
- `injectHardenRunnerPass` is **skipped** — harden-runner is a job-level concern; it belongs in the calling workflow, not the composite action
- `pinRunnersPass` is **skipped** — composite actions don't define `runs-on`

---

## Harden Runner Injection Pass

For each job, the pass checks if `step-security/harden-runner` is already the first step (pinned or unpinned). If present, that job is skipped.

If absent, a harden-runner step is prepended immediately after the `steps:` line, matching the surrounding indentation (same raw string substitution pattern used by `pinActionsPass`).

The injected block (egress-policy value comes from `p.egressPolicy`, i.e. `--egress-policy` flag):
```yaml
    - name: Harden the runner (Audit all outbound calls)
      uses: step-security/harden-runner@<resolved-sha> # <version>
      with:
        egress-policy: <p.egressPolicy>
```

The harden-runner action is **immediately pinned during injection**. The latest release tag is fetched via `GET /repos/step-security/harden-runner/releases/latest`, then `getCommitHashFromVersion("step-security/harden-runner", tag)` resolves it to a SHA using the existing API/git fallback path.

Returns: count of jobs where harden-runner was injected.

---

## Runner Label Pinning Pass

`resolveRunnerLabel(label string, customMap map[string]string) string` uses this resolution order:

1. **`--runner-map` flag** — user-supplied overrides checked first
2. **GitHub API** — `GET /repos/actions/runner-images/releases/latest` to dynamically infer current OS version from the release tag
3. **Hardcoded fallback** — used if API call fails (network error, rate limit, private environment):

```go
var defaultRunnerMap = map[string]string{
    "ubuntu-latest":  "ubuntu-24.04",
    "windows-latest": "windows-2022",
    "macos-latest":   "macos-15",
}
```

`pinRunnersPass` scans for `runs-on: <label>` patterns. Matrix expressions (`runs-on: ${{ matrix.os }}`) are **skipped** — cannot be statically resolved.

Replacement uses raw string substitution consistent with the existing approach.

Returns: count of runner labels replaced.

---

## Contextual Tips

After the summary block in `patchLocalRepository`, gha-pinner prints tips for features the user **did not** activate in the current run:

```
💡 Tips:
   • Use --inject-harden-runner to add step-security/harden-runner to every job (runtime supply chain protection)
   • Use --pin-runners to replace ubuntu-latest with a versioned runner label (e.g. ubuntu-24.04)
```

Rules:
- Only tips for unused features are shown.
- If all three features are active, no tips section is printed.
- Tips are printed once per repo at the `patchLocalRepository` summary level, not once per file.

---

## Summary Output Changes

Two new lines added to the existing `📊 Summary:` block (only shown when the respective flag is set):

```
   • Harden-runner injected: 5 job(s)
   • Runner labels pinned: 7
```

---

## Testing

| New test file | Coverage |
|---|---|
| `harden_runner_test.go` | Injection into jobs without it; skip if already present; skip for composite actions; SHA resolved before injection |
| `runner_pinning_test.go` | Resolution order (custom map → API → fallback); matrix expression skipped; `resolveRunnerLabel` unit tests |
| `composite_action_test.go` | `.github/actions/` files scanned; only `pinActionsPass` runs; `runs.steps` walked correctly |

The `WorkflowPatcher` struct enables direct unit testing — construct with specific fields, call `patchFile` on a temp file, assert output content.

---

## Files Changed

All changes land in `cmd/gha-pinner/main.go` plus three new test files. No new packages.

**Additions to `main.go`:**
- 4 new flag vars
- `WorkflowPatcher` struct + `patchFile` method (refactor of `processWorkflowFile`)
- `injectHardenRunnerPass` function
- `pinRunnersPass` function
- `resolveRunnerLabel` function
- Extended `patchLocalRepository` to scan `.github/actions/**/*.yml`
- Contextual tips logic in `patchLocalRepository`

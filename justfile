# resend: the SimplicityGuy/resend Terraform and OpenTofu provider (Go).
#
# `just` (no args) lists every recipe. CI runs the same recipes a developer
# does; it does not re-implement the gates.

# Every recipe line runs through `mise exec`, so it uses the tools pinned in
# mise.toml whether or not mise is activated in the calling shell (bh, CI, an
# IDE task runner).
set shell := ["mise", "exec", "--", "bash", "-euo", "pipefail", "-c"]

# Running `just` here already runs this repository's commands, so trusting its
# own mise.toml widens nothing. It lets a fresh clone (a CI checkout, a
# `bh work submit` clean checkout) run recipes before anyone has typed
# `mise trust`.
export MISE_TRUSTED_CONFIG_PATHS := if env("MISE_TRUSTED_CONFIG_PATHS", "") == "" { justfile_directory() } else { justfile_directory() + ":" + env("MISE_TRUSTED_CONFIG_PATHS") }

# tfplugindocs infers the provider name from the checkout's directory name,
# which is wrong in a worktree or a clone under another name, so both the
# registry name and the name rendered into page titles are explicit.
tfplugindocs_names := "--provider-name resend --rendered-provider-name terraform-provider-resend"

# Default: list the recipes.
_default:
    @just --list

# Only this repository's tools are named, so a global mise config that is not
# in this lockfile cannot fail the frozen install.
# Install the pinned tools from mise.lock (without updating it), the Go modules, and the pre-commit hook.
[group('setup')]
setup:
    #!/usr/bin/env bash
    set -euo pipefail
    mise trust --quiet "{{ justfile_directory() }}"
    mise install --locked $(mise ls --local --no-header | awk '{print $1}')
    mise exec -- go mod download
    if [ -z "$(git config --get core.hooksPath)" ]; then
      mise exec -- pre-commit install --install-hooks
    else
      echo "core.hooksPath is set; skipping 'pre-commit install'"
    fi

# Re-resolve mise.lock (and .mise/locks/) for the CI and operator platforms after editing a pin in mise.toml.
[group('setup')]
lock:
    mise lock --platform linux-x64,macos-arm64

# Offline: no Resend credentials and no API calls. The acceptance suite is `just testacc`.
# The authoritative pre-merge gate: hooks (format, golangci-lint, actionlint), vet, gosec, unit tests, build, release config, secret scan, registry docs drift.
[group('validate')]
check: lint vet sec test build release-check secret-scan docs-check

# Run every pre-commit hook over all files (gofmt/goimports, golangci-lint, actionlint, file hygiene, gitleaks on staged changes).
[group('validate')]
lint:
    pre-commit run --all-files --show-diff-on-failure

# Rewrite Go sources with the formatters in .golangci.yml (gofmt, goimports).
[group('develop')]
fmt:
    golangci-lint fmt

# Run go vet over every package.
[group('validate')]
vet:
    go vet ./...

# Run gosec over every package.
[group('validate')]
sec:
    gosec -quiet ./...

# TF_ACC is cleared so an exported value can never turn this into the acceptance run.
# Run the unit tests (acceptance tests skip without TF_ACC).
[group('validate')]
test:
    env -u TF_ACC go test ./...

# The 100% threshold is enforced by a later change; this only measures.
# Run the unit tests with a coverage profile (coverage.out) and print the per-function summary.
[group('validate')]
coverage:
    env -u TF_ACC go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Online and credentialed, so not part of `check`: it creates and destroys real Resend resources.
# Run the acceptance suite against the live Resend API (needs RESEND_API_KEY).
[group('validate')]
testacc:
    : "${RESEND_API_KEY:?testacc needs RESEND_API_KEY for a Resend account it may create and delete resources in}"
    TF_ACC=1 go test -v -count=1 -parallel=4 -timeout 120m ./internal/provider/

# Scan the full git history for secrets.
[group('validate')]
secret-scan:
    gitleaks git --redact --no-banner .

# Online (vuln.go.dev), so not part of `check`.
# Report known vulnerabilities in the module's dependencies and the Go standard library.
[group('validate')]
vulncheck:
    govulncheck ./...

# Validate .goreleaser.yml.
[group('validate')]
release-check:
    goreleaser check

# Build the provider binary (terraform-provider-resend, gitignored).
[group('build')]
build:
    go build -o terraform-provider-resend .

# Install the provider binary into GOBIN, for a Terraform dev_overrides block.
[group('build')]
install:
    go install .

# Regenerate the registry docs in docs/ from the provider schema, examples/ and templates.
[group('docs')]
docs:
    tfplugindocs generate {{ tfplugindocs_names }}

# Fails on a page tfplugindocs would add, change or remove, so a schema change
# without regenerated docs, or a resource without a page, fails `just check`.
# Fail if docs/ is invalid or differs from what `just docs` generates.
[group('docs')]
docs-check: docs
    tfplugindocs validate --provider-name resend
    @if [ -n "$(git status --porcelain -- docs)" ]; then git status --short -- docs; git diff --stat -- docs; echo "docs/ is stale: run 'just docs' and commit the result" >&2; exit 1; fi

# Online (downloads the Cloudflare provider), so not part of `check`. Builds the
# provider and points both CLIs at it with dev_overrides.
# Run `terraform validate` and `tofu validate` over every example in examples/.
[group('docs')]
validate-examples:
    scripts/validate-examples.sh

# Writes nothing: the next version, tag and changelog entry Commitizen would
# produce from .cz.toml. --yes answers the "first tag?" prompt so it runs
# unattended (CI, no tty) before v0.1.0 exists.
# Preview the next release bump without changing any file.
[group('release')]
bump-preview:
    cz bump --dry-run --yes

# Local only: it never pushes. Review the commit and tag, then push them deliberately.
# Bump the version, update CHANGELOG.md, commit and create the annotated v$version tag.
[group('release')]
bump:
    cz bump

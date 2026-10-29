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
# The authoritative pre-merge gate: hooks (format, golangci-lint, actionlint, zizmor), go.mod tidiness, vet, gosec, unit tests, build, release config, secret scan, registry docs drift, the supported-versions tables, and the examples under each pinned Terraform and OpenTofu.
[group('validate')]
check: lint tidy-check vet sec test build release-check secret-scan docs-check versions-check validate-examples

# Run every pre-commit hook over all files (gofmt/goimports, golangci-lint, actionlint, zizmor, file hygiene, gitleaks on staged changes).
[group('validate')]
lint:
    pre-commit run --all-files --show-diff-on-failure

# Rewrite Go sources with the formatters in .golangci.yml (gofmt, goimports).
[group('develop')]
fmt:
    golangci-lint fmt

# Writes nothing: `-diff` prints the edits `go mod tidy` would make and exits
# non-zero, so a missing, stale or wrongly `// indirect` requirement fails `check`.
# Fail if go.mod or go.sum differs from what `go mod tidy` produces.
[group('validate')]
tidy-check:
    go mod tidy -diff

# Run go vet over every package.
[group('validate')]
vet:
    go vet ./...

# Run gosec over every package.
[group('validate')]
sec:
    gosec -quiet ./...

# TF_ACC is cleared so an exported value can never turn this into the acceptance run.
# The profile is checked after the tests pass, so a test failure is reported as
# itself and coverage is only judged on a green run. No code is excluded from it.
# Run the unit tests (acceptance tests skip without TF_ACC) and fail below 100% statement coverage.
[group('validate')]
test: && _coverage-gate
    env -u TF_ACC go test -coverprofile=coverage.out ./...

# Every block in the profile must have run at least once; `go tool cover -func` rounds to one decimal, so it cannot tell 99.96% from 100%.
[private]
_coverage-gate:
    #!/usr/bin/env bash
    set -euo pipefail
    uncovered=$(awk 'NR > 1 && $NF == 0 { n++ } END { print n + 0 }' coverage.out)
    if [ "$uncovered" -ne 0 ]; then
      go tool cover -func=coverage.out | tail -n 1
      echo "coverage is below 100%: $uncovered uncovered block(s) in coverage.out; run 'just coverage' to see which functions" >&2
      exit 1
    fi
    echo "coverage: 100% of statements"

# Run the unit tests with a coverage profile (coverage.out) and print the per-function summary; `test` enforces the 100% threshold.
[group('validate')]
coverage:
    env -u TF_ACC go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Online and credentialed, so not part of `check`: it creates and destroys real Resend resources.
# The CLI and version come from mise.toml (scripts/tf-cli.sh): `latest` is the highest pinned version of that CLI, `oldest` the lowest, or name one exactly.
# TF_ACC_TERRAFORM_PATH points terraform-plugin-testing at that binary. It addresses the provider as registry.terraform.io/hashicorp/resend, which OpenTofu would not match (its registry is registry.opentofu.org), hence TF_ACC_PROVIDER_HOST.
# Run the acceptance suite against the live Resend API under a CLI: `just testacc`, `just testacc tofu`, `just testacc terraform oldest` (needs RESEND_API_KEY).
[group('validate')]
testacc cli="terraform" channel="latest":
    #!/usr/bin/env bash
    set -euo pipefail
    : "${RESEND_API_KEY:?testacc needs RESEND_API_KEY for a Resend account it may create and delete resources in}"
    path="$(scripts/tf-cli.sh path {{ quote(cli) }} {{ quote(channel) }})"
    export TF_ACC_TERRAFORM_PATH="$path"
    if [ {{ quote(cli) }} = tofu ]; then export TF_ACC_PROVIDER_HOST=registry.opentofu.org; else unset TF_ACC_PROVIDER_HOST; fi
    echo "acceptance suite under $("$path" version | head -n 1) ($path)"
    TF_ACC=1 go test -v -count=1 -parallel=4 -timeout 120m ./internal/provider/

# Online and credentialed. Deletes only objects whose name starts with `tf-test-`, the prefix every acceptance test uses.
# Remove leftover acceptance-test domains and API keys from the Resend account (needs RESEND_API_KEY).
[group('validate')]
sweep:
    : "${RESEND_API_KEY:?sweep needs RESEND_API_KEY for the Resend test account to clean up}"
    go test -v -count=1 -timeout 30m ./internal/provider/ -run '^$' -sweep=all

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

# Offline: it builds the provider and points the CLIs at it with dev_overrides, so no registry is involved. An example that needs another provider's schema (the Cloudflare one) is skipped; `validate-examples-online` covers it.
# Run `terraform validate` and `tofu validate` over the examples in examples/, under the oldest and the latest pinned version of each CLI.
[group('docs')]
validate-examples:
    scripts/validate-examples.sh

# Online: downloads the Cloudflare provider from the registries, so it is not part of `check`.
# Like `validate-examples`, but also validates the examples that use other providers.
[group('docs')]
validate-examples-online:
    scripts/validate-examples.sh --online

# Fails when README.md or templates/index.md.tmpl (the source of docs/index.md) does not name a version pinned in mise.toml, so the supported-versions tables cannot drift from the pins.
# Fail if a Terraform or OpenTofu version pinned in mise.toml is missing from the README or the docs.
[group('docs')]
versions-check:
    scripts/versions-check.sh

# Writes nothing: the next version, tag and changelog entry Commitizen would
# produce from .cz.toml, with the highlights in .release-notes/next.md inserted
# under the version heading as `just bump` will. Exits non-zero while the
# highlights are empty. --yes answers Commitizen's "first tag?" prompt so it
# runs unattended before v0.1.0 exists. Extra arguments go to `cz bump`.
# Preview the next release bump without changing any file.
[group('release')]
bump-preview *args:
    scripts/bump.sh --preview {{ args }}

# Local only: it never pushes. Review the commit and tag, then push them deliberately.
# Commitizen updates .cz.toml and CHANGELOG.md without committing, the highlights
# move from .release-notes/next.md into the new CHANGELOG section (the file is
# emptied), then one commit and the annotated v$version tag are made. Refuses
# while the highlights are empty. Extra arguments go to `cz bump`
# (e.g. `just bump --prerelease rc`).
# Bump the version, write the CHANGELOG entry with the highlights, commit and create the annotated v$version tag.
[group('release')]
bump *args:
    scripts/bump.sh {{ args }}

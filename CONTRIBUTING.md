# Contributing

Thanks for helping with the SimplicityGuy/resend provider. This guide covers the local setup, the checks every change must pass, and how releases are cut.

## Setup

Tool versions are pinned in `mise.toml` and locked in `mise.lock`. Install [mise](https://mise.jdx.dev), then:

```bash
just setup
```

This installs the pinned tools, downloads the Go modules and installs the pre-commit hook. `just --list` shows every recipe.

## The `just check` gate

`just check` is the pre-merge gate and must pass before a change is reviewed. It runs:

- the pre-commit hooks (gofmt/goimports, golangci-lint, actionlint, zizmor, file hygiene, gitleaks on staged changes)
- `go mod tidy -diff`, so `go.mod` and `go.sum` match what `go mod tidy` produces
- `go vet`
- gosec
- the unit tests
- the provider build
- the GoReleaser config check
- a gitleaks scan of the git history

Two further requirements are part of the gate (they are being wired into `just check`; treat them as required now):

- **100% test coverage.** Every statement in the provider must be covered by the unit tests. Run `just coverage` for the per-function summary and add tests for anything below 100%.
- **No docs drift.** The registry docs in `docs/` are generated from the provider schema, `examples/` and the templates. After changing a schema or an example, run `just docs` and commit the result. `just docs-check` fails when `docs/` is stale. Never edit generated files in `docs/` by hand.

Write tests first where you can. Acceptance tests live next to the unit tests in `internal/provider/`; the unit tests use a mock client and never touch the network.

## Acceptance tests

Acceptance tests create and delete real domains and API keys through the Resend API. They cover every resource (`resend_domain`, `resend_api_key`, `resend_domain_verification`) and data source (`resend_domain`, `resend_api_key`) the provider ships.

- Use a **dedicated TEST Resend account** and a `full_access` API key created in it. Never use a production account or a production key: the suite creates and deletes real objects.
- Export the key and run the suite (without `TF_ACC` and `RESEND_API_KEY`, `just testacc` refuses to start; `just test` never runs it):

```bash
export RESEND_API_KEY="re_..."
just testacc
```

- Test resources are named with the `tf-test-` prefix plus a random suffix and are destroyed by each test. After a failed or interrupted run, remove leftovers with `just sweep`. The sweeper deletes only domains and API keys whose name starts with `tf-test-`; give the key you run with a different name.
- Never print the key (no `set -x`, no `echo`), and keep it out of shell history and logs.

CI runs `just testacc` (then `just sweep`) on pushes to `main` and on pull requests from this repository, using the `RESEND_API_KEY` repository secret. It never runs on fork or Dependabot pull requests, which receive no secrets, and runs are serialized because they share one account.

## Terraform and OpenTofu

The provider targets both CLIs and will be published to both registries, so changes must work under both. Both are pinned in `mise.toml`. The acceptance suite uses whichever CLI the test framework finds; to run it under a specific one, set `TF_ACC_TERRAFORM_PATH` to that binary, for example:

```bash
TF_ACC_TERRAFORM_PATH="$(mise which tofu)" just testacc
TF_ACC_TERRAFORM_PATH="$(mise which terraform)" just testacc
```

A CI matrix that runs the suite against both CLIs doesn't exist yet; bead tfresend-ee6 adds it, and verifies and documents the exact OpenTofu setup.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org): `feat(domain): ...`, `fix(api_key): ...`, `docs: ...`, `chore: ...`, `build: ...`, `test: ...`. Write messages in English. The type drives the version bump: `feat` bumps the minor version and `fix` the patch while the provider is 0.x, and a `!` or `BREAKING CHANGE:` footer marks a breaking change. `docs`, `chore` and `test` commits do not bump the version.

## Releases

Releases are cut from `main` by a maintainer, with independent SemVer starting at v0.1.0. The plan for each release is in [docs/release-plan.md](docs/release-plan.md).

```bash
just bump-preview   # show the next version and changelog entry; writes nothing
just bump           # update CHANGELOG.md, commit, create the annotated vX.Y.Z tag (local only)
```

Check that `bump-preview` proposes the version the plan expects. After `just bump`, review the commit and tag, then push the tag deliberately; pushing a `v*` tag runs the release workflow, which publishes the signed assets.

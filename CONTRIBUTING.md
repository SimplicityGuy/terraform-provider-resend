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

- the pre-commit hooks (gofmt/goimports, golangci-lint, actionlint, file hygiene, gitleaks on staged changes)
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

Acceptance tests create and delete real domains and API keys through the Resend API.

- Use your own **dedicated test Resend account**. Never run them against a production account.
- Create an API key in that account and run:

```bash
export RESEND_API_KEY="re_..."
just testacc
```

Test resources use the `tf-test-` name prefix so the sweepers can remove leftovers.

## Terraform and OpenTofu

The provider is published for both CLIs, so changes must work under both. Both are pinned in `mise.toml`. The acceptance suite uses whichever CLI the test framework finds; to run it under a specific one, set `TF_ACC_TERRAFORM_PATH` (and `TF_ACC_PROVIDER_NAMESPACE=hashicorp` when using OpenTofu) to that binary, for example:

```bash
TF_ACC_TERRAFORM_PATH="$(mise which tofu)" just testacc
TF_ACC_TERRAFORM_PATH="$(mise which terraform)" just testacc
```

CI runs the suite against both.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org): `feat(domain): ...`, `fix(api_key): ...`, `docs: ...`, `chore: ...`, `build: ...`, `test: ...`. Write messages in English. The type drives the version bump: `feat` bumps the minor version and `fix` the patch while the provider is 0.x, and a `!` or `BREAKING CHANGE:` footer marks a breaking change. `docs`, `chore` and `test` commits do not bump the version.

## Releases

Releases are cut from `main` by a maintainer, with independent SemVer starting at v0.1.0. The plan for each release is in [doc/release-plan.md](doc/release-plan.md).

```bash
just bump-preview   # show the next version and changelog entry; writes nothing
just bump           # update CHANGELOG.md, commit, create the annotated vX.Y.Z tag (local only)
```

Check that `bump-preview` proposes the version the plan expects. After `just bump`, review the commit and tag, then push the tag deliberately; pushing a `v*` tag runs the release workflow, which publishes the signed assets.

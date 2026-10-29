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

- Domain tests create `tf-test-<random>.<RESEND_TEST_DOMAIN>`, so also set `RESEND_TEST_DOMAIN` to a parent domain that no other Resend account uses (Resend rejects a test domain under a domain another account has registered, such as `example.com`). Without it, the domain tests are skipped, not failed. In CI it comes from the `RESEND_TEST_DOMAIN` repository Actions variable.
- Test resources are named with the `tf-test-` prefix plus a random suffix and are destroyed by each test. After a failed or interrupted run, remove leftovers with `just sweep`. The sweeper deletes only domains and API keys whose name starts with `tf-test-`; give the key you run with a different name.
- Never print the key (no `set -x`, no `echo`), and keep it out of shell history and logs.

CI runs `just testacc` under Terraform and OpenTofu (then `just sweep`, see [Terraform and OpenTofu](#terraform-and-opentofu)) on pushes to `main` and on pull requests from this repository, using the `RESEND_API_KEY` repository secret. It never runs on fork or Dependabot pull requests, which receive no secrets, and runs are serialized because they share one account.

## Terraform and OpenTofu

The provider targets both CLIs and is published to both registries, so changes must work under both. The versions the provider supports are declared once, in the `terraform` and `opentofu` entries of `mise.toml` (checksummed in `mise.lock`):

| CLI       | Oldest supported | Latest tested |
| --------- | ---------------- | ------------- |
| Terraform | `1.0.11`         | `1.17.0`      |
| OpenTofu  | `1.6.3`          | `1.13.2`      |

The highest entry of each is the `terraform` / `tofu` on `PATH` and the "latest" leg; the lowest is the "oldest" leg. Everything else derives from them (`scripts/tf-cli.sh`): the CI matrix, `just testacc`, `just validate-examples`. The README and the docs index list the same versions, and `just versions-check` (part of `just check`) fails when they stop matching `mise.toml`. To raise or lower a version, edit `mise.toml`, run `just lock`, update the tables in `README.md` and `templates/index.md.tmpl`, run `just docs`, and commit them together.

Why these floors: the provider speaks plugin protocol 6 (the registry manifest declares it), which Terraform supports from 1.0, and 1.0.11 is the last 1.0 patch release. OpenTofu's first release is 1.6, and 1.6.3 is its last 1.6 patch release, so both floors are the oldest line a user can run with the newest fixes. Raise a floor only for a provider feature that needs a newer CLI, and say so in the docs. Config-driven `import` blocks need Terraform and OpenTofu 1.5+; that is the CLI's own limit, so the import examples are validated only where it applies and the `terraform import` command is documented beside them.

Run the acceptance suite under a specific CLI with `just testacc <terraform|tofu> [latest|oldest|VERSION]`, for example:

```bash
just testacc tofu            # the latest pinned OpenTofu
just testacc terraform oldest
```

It sets `TF_ACC_TERRAFORM_PATH` to that CLI's binary. For OpenTofu it also sets `TF_ACC_PROVIDER_HOST=registry.opentofu.org`: terraform-plugin-testing addresses the provider under `registry.terraform.io` by default, which `tofu` would not match. The examples are validated offline under all four versions by `just validate-examples` (part of `just check`), using a `dev_overrides` CLI config that points at the locally built provider; `just validate-examples-online` also validates the examples that use other providers, and CI runs it.

CI (`.github/workflows/acceptance.yml`) runs the four legs in order, then one sweep. Why it is a workflow of its own: the Resend account is shared, so only one run may touch it at a time, and the sweeper must never delete the live resources of a run still in flight. A concurrency group on the matrix jobs would cancel legs (GitHub keeps one pending job per group), so the group is on the workflow: a run holds it from its first leg to its sweep. The legs use `max-parallel: 1`, so the load on the account is that of one suite.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org): `feat(domain): ...`, `fix(api_key): ...`, `docs: ...`, `chore: ...`, `build: ...`, `test: ...`. Write messages in English. The type drives the version bump: `feat` bumps the minor version and `fix` the patch while the provider is 0.x, and a `!` or `BREAKING CHANGE:` footer marks a breaking change. `docs`, `chore` and `test` commits do not bump the version.

## Releases

Releases are cut from `main` by a maintainer, with independent SemVer starting at v0.1.0. The plan for each release is in [docs/release-plan.md](docs/release-plan.md).

```bash
just bump-preview   # show the next version and CHANGELOG section, highlights included; writes nothing
just bump           # update CHANGELOG.md, commit, create the annotated vX.Y.Z tag (local only)
```

Commitizen lists only `feat`, `fix` and `refactor` commits, so every release also needs hand-written highlights in `.release-notes/next.md`: what a user must know to upgrade or adopt it (a new address, a migration step, a dependency or supported-version change, signing). Write them in a bead, review them like code, and merge them before the bump; keep them to what `main` ships. `just bump` inserts them under the new version heading in `CHANGELOG.md`, empties the file, and refuses to run while it is empty. The release workflow publishes that CHANGELOG section as the GitHub release notes.

Check that `bump-preview` proposes the version the plan expects and shows the highlights as you want them. After `just bump`, review the commit and tag, then push the tag deliberately; pushing a `v*` tag runs the release workflow, which publishes the signed assets and fails if the tag has no CHANGELOG section.

### Release checks

After the Release workflow finishes, check the published assets before announcing the release. The
asset names are `terraform-provider-resend_X.Y.Z_*`.

1. Verify the signature, then the checksums, in an empty directory:

   ```bash
   gh release download vX.Y.Z --repo SimplicityGuy/terraform-provider-resend \
     --pattern '*_SHA256SUMS' --pattern '*_SHA256SUMS.sig' --pattern '*.zip' --pattern '*_manifest.json'
   gpg --import release-signing-key.asc
   gpg --verify terraform-provider-resend_X.Y.Z_SHA256SUMS.sig terraform-provider-resend_X.Y.Z_SHA256SUMS
   sha256sum --ignore-missing -c terraform-provider-resend_X.Y.Z_SHA256SUMS   # macOS: shasum -a 256 --ignore-missing -c <file>
   ```

   `gpg --verify` must report a good signature by `30D2732FD215D0089C27FCCB9F85682D8BB841F1`.
2. Once both registries have ingested the release, run `terraform init` and `tofu init` against a
   configuration that requires the provider. Set `OPENTOFU_ENFORCE_GPG_VALIDATION=true` for
   `tofu init` so OpenTofu refuses an unsigned or wrongly signed provider instead of warning. Both
   commands must report the signing key ID (`9F85682D8BB841F1`) in their output.

### Key rotation and revocation

The release key is RSA 4096 (the registries reject ECC keys) and expires on 2028-10-28. The private
key lives only in the operator's 1Password, the offline backup and the `release` environment
secrets (`GPG_PRIVATE_KEY`, `PASSPHRASE`; only `v*` tags can deploy to that environment). The
operator keeps the revocation certificate offline.

To rotate the key, for a planned replacement or before it expires (start by 2028-07):

1. Generate a new RSA 4096 key and a revocation certificate for it. Store both offline.
2. Reseal `GPG_PRIVATE_KEY` and `PASSPHRASE` into the `release` environment with `scripts/seal github`
   in the infra repository, then apply it.
3. Replace the public key in the Terraform Registry and the OpenTofu Registry (add the new key
   there; the registries verify each release against the keys registered for the namespace).
4. Replace `release-signing-key.asc`, and update the fingerprint, key ID and expiry in `README.md`,
   `SECURITY.md` and this file.
5. Cut the next release, run the release checks above, and keep the old public key available so
   earlier releases still verify.

To extend the key's expiry instead, change it with `gpg --edit-key`, re-export the public key,
update the expiry date here and in `SECURITY.md`, and update the registries' copy.

If the key is compromised or the private key is lost:

1. Revoke the key now: import the offline revocation certificate, export the revoked public key and
   commit it as `release-signing-key.asc`. Upload it to both registries.
2. Remove the old secrets from the `release` environment and rotate to a new key as above.
3. Notify users: publish a security advisory and a note in `SECURITY.md` and the README giving the
   revoked fingerprint, the new fingerprint, the first release signed with the new key, and which
   releases to re-verify or treat as suspect.

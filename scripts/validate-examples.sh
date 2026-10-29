#!/usr/bin/env bash
# Validate every example configuration under examples/ with each supported
# Terraform and OpenTofu (the lowest and the highest version pinned in
# mise.toml; see scripts/tf-cli.sh) against a locally built provider.
#
# The provider isn't on a registry yet, so a throwaway CLI config points the
# CLIs at the local build with dev_overrides. Examples that declare no
# required_providers block (most resource and data source snippets) get one for
# SimplicityGuy/resend, as a real root module would.
#
# By default this is offline, so `just check` can run it anywhere: an example
# that also uses another provider (the Cloudflare DNS wiring) can't be
# validated without that provider's schema, so it is skipped with a notice.
# `--online` validates those too, downloading the providers from the registry.
set -euo pipefail

online=0
case "${1:-}" in
  "") ;;
  --online) online=1 ;;
  *) echo "usage: ${0##*/} [--online]" >&2; exit 2 ;;
esac

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

go build -C "$root" -o "$work/bin/terraform-provider-resend" .

# dev_overrides makes both CLIs run the local build. `tofu init` still asks an
# installation method for every required provider, overridden or not, and the
# registries don't have this one yet, so the same build is also offered as a
# filesystem mirror (as version 0.1.0, which satisfies the examples' "~> 0.1")
# for init to find. Everything else, such as the Cloudflare provider, comes
# from the registry. The short source address resolves to each CLI's own
# default registry, so one file serves both.
target="$(go env GOOS)_$(go env GOARCH)"
for host in registry.terraform.io registry.opentofu.org; do
  mkdir -p "$work/mirror/$host/simplicityguy/resend/0.1.0/$target"
  cp "$work/bin/terraform-provider-resend" "$work/mirror/$host/simplicityguy/resend/0.1.0/$target/"
done
cat >"$work/cli.tfrc" <<CONFIG
provider_installation {
  dev_overrides {
    "simplicityguy/resend" = "$work/bin"
  }
  filesystem_mirror {
    path    = "$work/mirror"
    include = ["simplicityguy/resend"]
  }
  direct {
    exclude = ["simplicityguy/resend"]
  }
}
CONFIG
export TF_CLI_CONFIG_FILE="$work/cli.tfrc"
export TOFU_CLI_CONFIG_FILE="$work/cli.tfrc"
export TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1

"$("$root/scripts/tf-cli.sh" path terraform latest)" fmt -check -recursive -diff "$root/examples"

# Every pinned version of each CLI that is the lowest or the highest, once.
cases=()
for cli in terraform tofu; do
  oldest="$("$root/scripts/tf-cli.sh" version "$cli" oldest)"
  latest="$("$root/scripts/tf-cli.sh" version "$cli" latest)"
  cases+=("$cli $oldest")
  [ "$oldest" = "$latest" ] || cases+=("$cli $latest")
done

dirs=()
while IFS= read -r dir; do dirs+=("$dir"); done < <(cd "$root" && find examples -name '*.tf' -exec dirname {} \; | sort -u)

failed=0
for entry in "${cases[@]}"; do
  read -r cli version <<<"$entry"
  bin="$("$root/scripts/tf-cli.sh" path "$cli" "$version")"
  for dir in "${dirs[@]}"; do
    label="$cli $version $dir"
    if [ "$online" -eq 0 ] && grep -hE '^[[:space:]]*source[[:space:]]*=' "$root/$dir"/*.tf | grep -qvF '"SimplicityGuy/resend"'; then
      echo "skip  $label (needs another provider's schema; run 'just validate-examples-online')"
      continue
    fi
    module="$work/$cli-$version/$dir"
    mkdir -p "$module"
    cp "$root/$dir"/*.tf "$module/"
    # `import` blocks (documented as Terraform and OpenTofu 1.5+) are a language
    # feature, not the provider's: Terraform 1.0 rejects them whatever the provider
    # does. Leave just those files out for Terraform before 1.5, as the docs do.
    if [ "$cli" = terraform ] && [ "$(printf '%s\n1.5.0\n' "$version" | sort -V | head -n 1)" != 1.5.0 ]; then
      { grep -lE '^import[[:space:]]*\{' "$module"/*.tf || true; } | xargs -r rm -f --
      if [ -z "$(ls "$module"/*.tf 2>/dev/null)" ]; then
        echo "skip  $label (only an import block, which needs 1.5+)"
        continue
      fi
    fi
    if ! grep -qs required_providers "$module"/*.tf; then
      cat >"$module/zz_required_providers.tf" <<'TF'
terraform {
  required_providers {
    resend = {
      source = "SimplicityGuy/resend"
    }
  }
}
TF
    fi
    if "$bin" -chdir="$module" init -backend=false -input=false >"$module.init.log" 2>&1 &&
      "$bin" -chdir="$module" validate -no-color >"$module.validate.log" 2>&1; then
      echo "ok    $label"
    else
      echo "FAIL  $label"
      cat "$module.init.log" "$module.validate.log" 2>/dev/null || true
      failed=1
    fi
  done
done
exit "$failed"

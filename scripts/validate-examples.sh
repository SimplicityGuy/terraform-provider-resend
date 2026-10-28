#!/usr/bin/env bash
# Validate every example configuration under examples/ with Terraform and
# OpenTofu against a locally built provider.
#
# The provider isn't on a registry yet, so a throwaway CLI config points both
# CLIs at the local build with dev_overrides. Examples that declare no
# required_providers block (most resource and data source snippets) get one for
# SimplicityGuy/resend, as a real root module would. Examples that use other
# providers (the Cloudflare DNS wiring) download them from the registry, so
# this needs network access.
set -euo pipefail

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

terraform fmt -check -recursive -diff "$root/examples"

mapfile -t dirs < <(cd "$root" && find examples -name '*.tf' -exec dirname {} \; | sort -u)
failed=0
for cli in terraform tofu; do
  for dir in "${dirs[@]}"; do
    module="$work/$cli/$dir"
    mkdir -p "$module"
    cp "$root/$dir"/*.tf "$module/"
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
    if "$cli" -chdir="$module" init -backend=false -input=false >"$module.init.log" 2>&1 &&
      "$cli" -chdir="$module" validate -no-color >"$module.validate.log" 2>&1; then
      echo "ok    $cli $dir"
    else
      echo "FAIL  $cli $dir"
      cat "$module.init.log" "$module.validate.log" 2>/dev/null || true
      failed=1
    fi
  done
done
exit "$failed"

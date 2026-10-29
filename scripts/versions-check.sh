#!/usr/bin/env bash
# Fail if a Terraform or OpenTofu version pinned in mise.toml (the one place the
# supported versions are declared) is not named in the README and in the docs
# index. docs/index.md is generated from templates/index.md.tmpl, so the template
# and the generated page are both checked.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
failed=0
for cli in terraform tofu; do
  while IFS= read -r version; do
    for file in README.md templates/index.md.tmpl docs/index.md; do
      if ! grep -qF "\`$version\`" "$root/$file"; then
        echo "$file does not list $cli \`$version\`, pinned in mise.toml" >&2
        failed=1
      fi
    done
  done < <("$root/scripts/tf-cli.sh" versions "$cli")
done
exit "$failed"

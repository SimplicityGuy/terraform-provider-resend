#!/usr/bin/env bash
# Print the body of one release's CHANGELOG.md section: everything under the
# `## <tag> (<date>)` heading, up to the next `## ` heading. The heading itself
# is left out because the GitHub release already carries the tag as its title.
#
# This is the release notes GoReleaser publishes (release.yml passes the output
# to --release-notes). It exits non-zero when the section is missing or blank,
# so a tag cut without `just bump` fails the release instead of publishing
# empty notes.
#
# Usage: scripts/changelog-section.sh <tag> [changelog-file]   e.g. v0.1.0
set -euo pipefail

tag="${1:?usage: changelog-section.sh <tag> [changelog-file]}"
changelog="${2:-$(dirname "$0")/../CHANGELOG.md}"

section=$(awk -v tag="$tag" '
  /^## / {
    if (inside) exit
    # Commitizen writes "## v0.1.0 (2026-10-28)"; match the whole tag, so v0.1
    # never selects v0.1.0.
    inside = index($0, "## " tag " ") == 1 || $0 == "## " tag
    next
  }
  inside { print }
' "$changelog")

if [ -z "$(tr -d '[:space:]' <<<"$section")" ]; then
  echo "no CHANGELOG.md section for ${tag}, or it is empty; cut releases with 'just bump'" >&2
  exit 1
fi

# Trim the blank lines around the section; the caller owns the surrounding text.
printf '%s\n' "$section" | awk 'NF { started = 1 } started { lines[++n] = $0 } END { while (n && lines[n] ~ /^[[:space:]]*$/) n--; for (i = 1; i <= n; i++) print lines[i] }'

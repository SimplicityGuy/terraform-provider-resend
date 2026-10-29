#!/usr/bin/env bash
# Cut a release locally: Commitizen computes the version and the generated
# CHANGELOG entry, and the hand-written highlights in .release-notes/next.md
# are inserted under the new version heading, above the generated entries.
#
#   scripts/bump.sh --preview [cz bump options]   show the next release; writes nothing
#   scripts/bump.sh [cz bump options]             bump, commit and tag; never pushes
#
# Commitizen runs with --files-only so the highlights can join the same commit
# and tag; the commit message and the annotated tag are the ones `cz bump`
# would have made ("bump: version A → B", tag message "vB").
set -euo pipefail

cd "$(dirname "$0")/.."
highlights=.release-notes/next.md

preview=false
if [ "${1:-}" = "--preview" ]; then
  preview=true
  shift
fi

has_highlights() {
  [ -f "$highlights" ] && [ -n "$(tr -d '[:space:]' <"$highlights")" ]
}

# Print stdin with the highlights inserted after the first `## ` heading, which
# is the new version's.
insert_highlights() {
  HIGHLIGHTS=$(cat "$highlights") awk '
    { print }
    !done && /^## / { print ""; print ENVIRON["HIGHLIGHTS"]; done = 1 }
  '
}

if ! has_highlights; then
  echo "${highlights} is empty: write this release's highlights there (CONTRIBUTING.md, Releases) and merge them before bumping" >&2
fi

if $preview; then
  # The dry run prints three status lines, a blank line, then the section.
  output=$(cz bump --dry-run --yes "$@")
  printf '%s\n' "$output" | awk '/^## /{exit} {print}'
  printf '%s\n' "$output" | awk '/^## /{p=1} p' | insert_highlights
  has_highlights
  exit
fi

has_highlights || exit 1

if [ -n "$(git status --porcelain)" ]; then
  echo "the working tree is not clean; bump from a clean checkout of main" >&2
  exit 1
fi

old=$(cz version --project)
trap 'git checkout -- .cz.toml CHANGELOG.md "$highlights"' ERR
cz bump --files-only "$@"
new=$(cz version --project)

insert_highlights <CHANGELOG.md >CHANGELOG.md.new
mv CHANGELOG.md.new CHANGELOG.md
: >"$highlights"

git add .cz.toml CHANGELOG.md "$highlights"
git commit -m "bump: version ${old} → ${new}"
trap - ERR
git tag -a "v${new}" -m "v${new}"
echo "created v${new}; review the commit and tag, then push them deliberately"

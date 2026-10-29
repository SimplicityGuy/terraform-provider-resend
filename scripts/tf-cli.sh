#!/usr/bin/env bash
# Resolve the Terraform and OpenTofu versions this provider supports.
#
# The versions are declared once, in the `terraform` and `opentofu` entries of
# mise.toml (and checksummed in mise.lock). Everything that needs them, such as
# `just testacc`, `just validate-examples`, the CI matrix and `just
# versions-check`, asks this script instead of repeating them.
#
#   tf-cli.sh versions <terraform|tofu>             every supported version, oldest first
#   tf-cli.sh version  <terraform|tofu> <channel>   the version a channel names
#   tf-cli.sh path     <terraform|tofu> <channel>   the installed binary for that channel
#
# A channel is `latest` (the highest supported version), `oldest` (the lowest),
# or an exact version from mise.toml.
set -euo pipefail

usage() {
  echo "usage: ${0##*/} versions <terraform|tofu> | version|path <terraform|tofu> <latest|oldest|VERSION>" >&2
  exit 2
}

# The CLI's name is also its binary; OpenTofu's mise tool is called opentofu.
tool_of() {
  case "$1" in
    terraform) echo terraform ;;
    tofu) echo opentofu ;;
    *) echo "unknown CLI '$1': expected terraform or tofu" >&2; exit 2 ;;
  esac
}

versions() {
  local tool
  tool="$(tool_of "$1")" || exit
  mise ls --local --no-header "$tool" | awk '{print $2}' | sort -V
}

version() {
  local cli=$1 channel=$2 all
  all="$(versions "$cli")" || exit
  [ -n "$all" ] || { echo "mise.toml pins no version of $cli" >&2; exit 1; }
  case "$channel" in
    latest) tail -n 1 <<<"$all" ;;
    oldest) head -n 1 <<<"$all" ;;
    *)
      grep -Fxq -- "$channel" <<<"$all" || {
        echo "$cli $channel is not pinned in mise.toml (pinned: $(tr '\n' ' ' <<<"$all"))" >&2
        exit 1
      }
      echo "$channel"
      ;;
  esac
}

[ $# -ge 2 ] || usage
case "$1" in
  versions) versions "$2" ;;
  version) [ $# -eq 3 ] || usage; version "$2" "$3" ;;
  path)
    [ $# -eq 3 ] || usage
    # A variable, not a nested $(...): a failure there would not stop the script.
    v="$(version "$2" "$3")" || exit
    tool="$(tool_of "$2")" || exit
    mise which "$2" --tool "$tool@$v"
    ;;
  *) usage ;;
esac

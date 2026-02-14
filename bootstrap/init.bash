#!/usr/bin/env bash

set -e

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --no-binary)
    NO_BINARY=1
    shift
    ;;
  --no-versioning)
    NO_VERSIONING=1
    shift
    ;;
  -*)
    echo "Unknown option $1"
    exit 1
    ;;
  *)
    POSITIONAL_ARGS+=("$1")
    shift
    ;;
  esac
done

set -- "${POSITIONAL_ARGS[@]}"

if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Usage: $0 <github-account-name> <repo-name>" >&2
  exit 1
fi

ACCOUNT_NAME=$1
REPO_NAME=$2

if [[ $NO_BINARY ]]; then
  echo "--no-binary flag is set, removing related files" >&2
  rm -rf .goreleaser.yml .github/workflows/release.yml

  if [[ ! -f justfile ]]; then
    echo "ERROR: justfile not found, cannot remove binary recipes" >&2
    exit 1
  fi

  awk '
    /^# Build .* binary$/ { in_binary=1; next }
    /^# Build and release/ { in_release=1; next }
    in_binary && /^[^ \t#]/ { in_binary=2 }
    in_release && /^[^ \t#]/ { in_release=2 }
    in_binary == 2 && /^$/ { in_binary=0; next }
    in_release == 2 && /^$/ { in_release=0; next }
    !in_binary && !in_release { print }
  ' justfile >justfile_cp

  if [[ ! -s justfile_cp ]]; then
    echo "ERROR: AWK script produced empty justfile, this indicates a logic error" >&2
    rm -f justfile_cp
    exit 1
  fi

  if ! grep -q '^[a-z].*:' justfile_cp; then
    echo "ERROR: Processed justfile appears invalid (no recipes found)" >&2
    echo "Original justfile preserved, removing failed copy" >&2
    rm -f justfile_cp
    exit 1
  fi

  mv justfile_cp justfile
else
  mv cmd/x-repo-name "cmd/$REPO_NAME"
fi

if [[ $NO_VERSIONING ]]; then
  echo "--no-versioning flag is set, removing related files" >&2
  rm -rf \
    .github/scripts/release-notes.bash \
    .github/release-drafter.yml \
    .github/workflows/release-drafter.yml
fi

grep -rl x-github-account-name . 2>/dev/null |
  xargs -r sed -i "s/x-github-account-name/$ACCOUNT_NAME/g" || true
grep -rl x-repo-name . 2>/dev/null |
  xargs -r sed -i "s/x-repo-name/$REPO_NAME/g" || true

rm -rf bootstrap test gitsync.json

echo -e "# $REPO_NAME\n\nTODO" >README.md

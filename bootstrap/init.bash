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
  rm -rf .goreleaser.yml .github/workflows/release.yml bin
  awk '/\.PHONY: (build|release)/ {d=1}; !d {print}; /^$/ {d=0}' Makefile > Makefile_cp && Makefile_cp > Makefile
else
  mv cmd/x-repo-name "cmd/$REPO_NAME"
fi

if [[ $NO_VERSIONING ]]; then
  echo "--no-versioning flag is set, removing related files" >&2
  rm -rf scripts/release-notes.sh release-drafter.yml workflows/release-drafter.yml
fi

grep -rl x-github-account-name | xargs sed -i "s/x-github-account-name/$ACCOUNT_NAME/g"
grep -rl x-repo-name | xargs sed -i "s/x-repo-name/$REPO_NAME/g"
rm -rf bootstrap
rm gitsync.json

echo -e "# $REPO_NAME\nTODO\n" > README.md

#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
: "${MYGO_UPDATER_PRIVATE_KEY:?Set the EdenFork update signing key}"
: "${SEE_SIGNING_IDENTITY:?Set the Developer ID Application identity}"
: "${SEE_NOTARY_PROFILE:?Set the notarytool keychain profile}"

source_dir=$(mktemp -d)
trap 'rm -rf "$source_dir"' EXIT
git archive HEAD | tar -x -C "$source_dir"
version=$(jq -er '.version | select(endswith(".eden"))' "$source_dir/mygo.json")
output="$PWD/build/release-$version"

cd "$source_dir"
jq --arg profile "$SEE_NOTARY_PROFILE" \
  '.macos.notarize = {keychainProfile: $profile}' mygo.json > mygo.release.json
mv mygo.release.json mygo.json
go tool mygo build -platform darwin/arm64 -sign "$SEE_SIGNING_IDENTITY" -o "$output"
test -s "$output/darwin-arm64/update-darwin-arm64.json"

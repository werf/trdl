#!/bin/bash
set -euo pipefail

script_dir="$(cd "$( dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
project_dir="$script_dir/.."

version="${1:?Version should be set}"

declare -A regexps
regexps["$project_dir/dist/$version/linux-amd64/bin/vault-plugin-secrets-trdl"]="x86-64.*statically linked"
regexps["$project_dir/dist/$version/linux-arm64/bin/vault-plugin-secrets-trdl"]="ARM aarch64.*statically linked"

for filename in "${!regexps[@]}"; do
  if ! [[ -f "$filename" ]]; then
    echo Binary at "$filename" does not exist.
    exit 1
  fi

  file "$filename" | awk -v regexp="${regexps[$filename]}" '{print $0; if ($0 ~ regexp) { exit } else { print "Unexpected binary info ^^"; exit 1 }}'
done
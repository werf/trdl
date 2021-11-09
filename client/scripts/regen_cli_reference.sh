#!/bin/bash

set -e

SOURCE_PATH="$(realpath "${BASH_SOURCE[0]}")"
PROJECT_DIR="$(dirname $SOURCE_PATH)/../.."

function regen() {
  saved_dir="$PWD"
  cd "$PROJECT_DIR"/client
  go install github.com/werf/trdl/client/cmd/trdl

  # regen CLI partials, pages and sidebar
  trdl docs ../docs

  cd "$saved_dir"
}

function generate_sidebar() {
  reference_sidebar_path="$PROJECT_DIR/docs/_data/sidebars/reference.yml"
  cli_partial_path="$PROJECT_DIR/docs/_data/sidebars/_cli.yml"
  reference_partial_path="$PROJECT_DIR/docs/_data/sidebars/_reference.yml"

  cat << EOF > "$reference_sidebar_path"
# This file is generated by "client/scripts/regen_cli_reference.sh".
# DO NOT EDIT!

# This is your sidebar TOC. The sidebar code loops through sections here and provides the appropriate formatting.

EOF

  cat "$cli_partial_path" >> "$reference_sidebar_path"
  echo -e "\n" >> "$reference_sidebar_path"
  cat "$reference_partial_path" >> "$reference_sidebar_path"
}

regen
generate_sidebar
#!/bin/bash

#
# Generate the CRD document.
#

set -eo pipefail

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."
readonly source_path="${topd}/api/v1"
readonly output_path="${topd}/docs"

docs() {
  "${d}/tool.sh" crd-ref-docs "$@"
}

gomodver() {
  local -r __name="$1"
  go -C "$topd" list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' "$1" 2>/dev/null
}

k8s_version() {
  gomodver "sigs.k8s.io/controller-runtime"
}

# Needed to link to the standard resource documentation for a specific Kubernetes version.
generate_config() {
    cat <<EOS
render:
  kubernetesVersion: $(k8s_version)
EOS
}


mkdir -p "$output_path"
config="$(mktemp)"
generate_config > "$config"

docs --source-path="$source_path" \
     --output-path="$output_path" \
     --config="$config" \
     --output-mode=group \
     --renderer markdown

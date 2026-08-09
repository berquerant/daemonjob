#!/bin/bash

#
# Update Kubernetes dependencies (k8s.io/*, controller-runtime, etc.) to target version.
# Automatically detects current k8s.io/* dependencies in go.mod without hardcoding.
#
# Usage:
#   ./hack/update-k8s-deps.sh <k8s-version> <controller-runtime-version> [controller-tools-version]
#
# Example:
#   ./hack/update-k8s-deps.sh 1.31.2 0.19.3 v0.16.5
#

set -eo pipefail

log() {
  echo >&2 "${0##*/}: $*"
}

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

readonly k8s_ver="$1"
readonly cr_ver="$2"
readonly ct_ver="$3"

if [[ -z "$k8s_ver" || -z "$cr_ver" ]]; then
  cat <<EOS >&2
Usage: ${0##*/} <k8s-version> <controller-runtime-version> [controller-tools-version]

  k8s-version:                e.g. 1.31.2 or v1.31.2
  controller-runtime-version: e.g. 0.19.3 or v0.19.3
  controller-tools-version:   e.g. v0.16.5 (optional)

Examples:
  ${0##*/} 1.31.2 0.19.3 v0.16.5
EOS
  exit 1
fi

# Normalize versions (strip 'v' prefix for parsing minor version)
readonly clean_k8s_ver="${k8s_ver#v}"
readonly clean_cr_ver="${cr_ver#v}"

# Extract minor version for k8s.io submodules (e.g. 1.31.2 -> 0.31.2)
readonly k8s_minor_ver="$(echo "$clean_k8s_ver" | sed -E 's/^[0-9]+\.([0-9]+\.[0-9]+)/0.\1/')"

log "Target Kubernetes version: v${clean_k8s_ver}"
log "Target k8s.io module version: v${k8s_minor_ver}"
log "Target controller-runtime version: v${clean_cr_ver}"

if [[ -n "$ct_ver" ]]; then
  readonly clean_ct_ver="v${ct_ver#v}"
  log "Target controller-tools version: ${clean_ct_ver}"
  
  readonly env_file="${topd}/.github/actions/setup-env/action.yml"
  if [[ -f "$env_file" ]]; then
    log "Updating CONTROLLER_GEN_VERSION and KUBECTL_VERSION in ${env_file}..."
    sed -i '' -E "s/KUBECTL_VERSION=v[0-9]+\.[0-9]+\.[0-9]+/KUBECTL_VERSION=v${clean_k8s_ver}/g" "$env_file"
    sed -i '' -E "s/CONTROLLER_GEN_VERSION=v[0-9]+\.[0-9]+\.[0-9]+/CONTROLLER_GEN_VERSION=${clean_ct_ver}/g" "$env_file"
  fi
fi

# Automatically detect current k8s.io/* dependencies from go.mod (excluding k8s.io/kubernetes if any)
log "Detecting active k8s.io/* dependencies in go.mod..."
mapfile -t k8s_modules < <(go -C "$topd" list -m all | awk '{print $1}' | grep -E '^k8s\.io/' | grep -v '^k8s\.io/kubernetes$' || true)

if [[ ${#k8s_modules[@]} -eq 0 ]]; then
  log "Warning: No k8s.io/* modules found in go.mod!"
else
  log "Found ${#k8s_modules[@]} k8s.io submodule(s): ${k8s_modules[*]}"
fi

targets=("sigs.k8s.io/controller-runtime@v${clean_cr_ver}")
for mod in "${k8s_modules[@]}"; do
  targets+=("${mod}@v${k8s_minor_ver}")
done

log "Updating Go modules..."
go -C "$topd" get "${targets[@]}"

log "Running go mod tidy..."
go -C "$topd" mod tidy

log "Successfully updated Kubernetes Go dependencies to v${clean_k8s_ver}!"
log "Remember to run 'make generate', 'make manifests', 'make build-installer', 'make chart', 'make docs', and 'make test-e2e' to complete code & manifest sync."

#!/bin/bash

#
# Update Kubernetes dependencies (k8s.io/*, controller-runtime, etc.) to target version.
# Dynamically detects current v0.Y.Z versioned k8s.io/* dependencies directly from go.mod.
#
# Usage:
#   ./hack/update-k8s-deps.sh <k8s-version> <controller-runtime-version> [controller-tools-version] [k0s-version]
#
# Example:
#   ./hack/update-k8s-deps.sh 1.36.3 0.24.1 v0.21.0 v1.36.2-k0s.0
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
readonly k0s_ver="$4"

if [[ -z "$k8s_ver" || -z "$cr_ver" ]]; then
  cat <<EOS >&2
Usage: ${0##*/} <k8s-version> <controller-runtime-version> [controller-tools-version] [k0s-version]

  k8s-version:                e.g. 1.36.3 or v1.36.3
  controller-runtime-version: e.g. 0.24.1 or v0.24.1
  controller-tools-version:   e.g. v0.21.0 (optional)
  k0s-version:                e.g. v1.36.2-k0s.0 (optional)

Examples:
  ${0##*/} 1.36.3 0.24.1 v0.21.0 v1.36.2-k0s.0
EOS
  exit 1
fi

# Normalize versions (strip 'v' prefix for parsing minor version)
readonly clean_k8s_ver="${k8s_ver#v}"
readonly clean_cr_ver="${cr_ver#v}"

# Extract minor version for k8s.io submodules (e.g. 1.36.3 -> 0.36.3)
readonly k8s_minor_ver="$(echo "$clean_k8s_ver" | sed -E 's/^[0-9]+\.([0-9]+\.[0-9]+)/0.\1/')"

log "Target Kubernetes version: v${clean_k8s_ver}"
log "Target k8s.io module version: v${k8s_minor_ver}"
log "Target controller-runtime version: v${clean_cr_ver}"

readonly env_file="${topd}/.github/actions/setup-env/action.yml"
if [[ -f "$env_file" ]]; then
  log "Updating KUBECTL_VERSION in ${env_file}..."
  sed -i '' -E "s/KUBECTL_VERSION=v[0-9]+\.[0-9]+\.[0-9]+/KUBECTL_VERSION=v${clean_k8s_ver}/g" "$env_file"

  if [[ -n "$ct_ver" ]]; then
    readonly clean_ct_ver="v${ct_ver#v}"
    log "Updating CONTROLLER_GEN_VERSION in ${env_file} to ${clean_ct_ver}..."
    sed -i '' -E "s/CONTROLLER_GEN_VERSION=v[0-9]+\.[0-9]+\.[0-9]+/CONTROLLER_GEN_VERSION=${clean_ct_ver}/g" "$env_file"
  fi

  if [[ -n "$k0s_ver" ]]; then
    readonly clean_k0s_ver="${k0s_ver}"
    log "Updating K0S_VERSION in ${env_file} to ${clean_k0s_ver}..."
    sed -i '' -E "s/K0S_VERSION=v[0-9]+\.[0-9]+\.[0-9]+-k0s\.[0-9]+/K0S_VERSION=${clean_k0s_ver}/g" "$env_file"
  fi
fi

# Dynamically parse v0.Y.Z versioned k8s.io submodules directly from go.mod
log "Detecting active k8s.io/* submodules in go.mod..."
readonly k8s_gomod_file="${topd}/go.mod"
k8s_modules=()
while read -r mod; do
  if [[ -n "$mod" ]]; then
    k8s_modules+=("$mod")
  fi
done < <(grep -E '^\s*k8s\.io/' "$k8s_gomod_file" | awk '{print $1}' | grep -E '^k8s\.io/(api|apimachinery|client-go|apiextensions-apiserver|apiserver|component-base|component-helpers|controller-manager|kubelet|kms|cri-api)' || true)

if [[ ${#k8s_modules[@]} -eq 0 ]]; then
  log "Warning: No k8s.io/* submodules found in go.mod!"
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

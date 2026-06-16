#!/bin/bash

#
# Install and invoke tools for development.
#

set -eo pipefail

log() {
  echo >&2 "${0##*/}: $*"
}

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

readonly name="$1"
if [[ -z "$name" ]] ; then
  log "No tool name!"
  exit 1
fi
shift

readonly bind="${topd}/bin"
readonly tmpd="${topd}/tmp"
case "$name" in
  "clean")
    log "Clean all binaries"
    find "$bind" -type f -maxdepth 1 -delete
    exit
    ;;
esac
mkdir -p "$bind" "$tmpd"

readonly bin="${bind}/${name}"

gomodver() {
  local -r __name="$1"
  go -C "$topd" list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' "$1" 2>/dev/null
}

# The version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31).
envtest_k8s_version() {
  gomodver "k8s.io/api" |  sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/'
}

use_envtest() {
  log "Setting up envtest binaries for Kubernetes version $(envtest_k8s_version)..."
  "${bin}" use "$(envtest_k8s_version)" --bin-dir "${bind}" -p path || {
    log "Error: Failed to set up envtest binaries for version $(envtest_k8s_version)."
    exit 1
  }
}

setup() {
  "${d}/setup.sh" "$name" "$bin" "$@"
}

setup_go() {
  "${d}/setup-go.sh" "$name" "$bind" "$@"
}

if [[ ! -x "$bin" ]] ; then
  log "Install ${bin} ..."
  case "$name" in
    "kubectl") setup "$KUBECTL_VERSION" ;;
    "setup-envtest") setup_go ;;
    "controller-gen") setup_go "$CONTROLLER_GEN_VERSION" ;;
    "kustomize") setup_go "$KUSTOMIZE_VERSION" ;;
    "golangci-lint") setup_go "$GOLANGCI_LINT_VERSION" ;;
    "go-licenses") setup_go "$GO_LICENSES_VERSION" ;;
    "helmify") setup_go "$HELMIFY_VERSION" ;;
    "crd-ref-docs") setup_go "$CRD_REF_DOCS_VERSION" ;;
    "helm") setup "$HELM_VERSION" ;;
    "helm-schema") setup_go "$HELM_SCHEMA_VERSION" ;;
    "yq") setup "$YQ_VERSION" ;;
    "shellcheck") setup "$SHELLCHECK_VERSION" ;;
    *)
      log "Unknown tool!: ${name}"
      exit 1
      ;;
  esac
fi

case "$name" in
  "setup-envtest") "$bin" use "$(envtest_k8s_version)" --bin-dir "$bind" -p path ;;
  *) "$bin" "$@" ;;
esac

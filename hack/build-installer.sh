#!/bin/bash

#
# Build the all-in-one manifest and the helm chart.
#

set -eo pipefail

readonly manifest_dir="manifests"
readonly manifest="${manifest_dir}/install.yaml"

readonly chart_name="daemonjob"
readonly chart_dir="charts"
readonly chart="${chart_dir}/${chart_name}"
readonly chart_yaml="${chart}/Chart.yaml"

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

cd "$topd"

tool() {
  "${topd}/hack/tool.sh" "$@"
}

kustomize() {
  tool kustomize "$@"
}

helmify() {
  tool helmify "$@"
}

helm() {
  tool helm "$@"
}

helm_schema() {
  tool helm-schema "$@"
}

build_manifest() {
  mkdir -p "$manifest_dir"
  make deploy-kustomize-manager
  tool kustomize build config/default > "$manifest"
}

generate_chart_yaml() {
  local -r __version="$1"
  if [[ -z "$__version" ]] ; then
    echo >&2 "No version!"
    return 1
  fi
  echo >&2 "Version: ${__version}"
  cat <<EOS
apiVersion: v2
name: daemonjob
description: Provides Job CRDs for managing cluster-wide node-local workloads
type: application
version: "${__version}"
appVersion: "${__version}"
kubeVersion: ">= 1.35.0-0"
keywords:
  - jobs
  - cronjobs
home: https://github.com/berquerant/daemonjob
maintainers:
  - name: berquerant
EOS
}

generate_values_schema() {
  pushd "$chart" > /dev/null
  helm_schema
  popd > /dev/null
}

build_chart() {
  local -r __version="$1"
  rm -rf "$chart_dir"
  helmify "$chart" < "$manifest"
  generate_chart_yaml "$__version" > "$chart_yaml"
  generate_values_schema
  helm lint --strict "$chart"
}

readonly cmd="$1"
case "$cmd" in
  "manifest") build_manifest ;;
  "chart")
    shift
    build_chart "$@"
    ;;
  *)
    cat <<EOS >&2
Unknown command: ${cmd}
Available commands are following:
  manifest
    Generate a consolidated YAML with CRDs and deployment into ${manifest}.
  chart VERSION
    Generate chart into ${chart}.
EOS
    exit 1
  ;;
esac

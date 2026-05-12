#!/bin/bash
set -eo pipefail

readonly self_name="$SELF_NAME"
readonly namespace="$NAMESPACE"
readonly daemon_job_name="$DAEMON_JOB_NAME"
readonly daemon_cronjob_name="$DAEMON_CRONJOB_NAME"
readonly controller_uid="$CONTROLLER_UID"
readonly base_job_name="base-job"

readonly daemon_job_role_label="daemonjob.berquerant.github.io/role"
readonly daemon_job_role_value="worker"
readonly daemon_job_name_label="daemonjob.berquerant.github.io/daemonjob-name"
readonly daemon_cronjob_name_label="daemonjob.berquerant.github.io/daemoncronjob-name"
readonly daemon_job_node_label="daemonjob.berquerant.github.io/node"

log() {
  echo >&2 "$*"
}

# Write termination logs.
# https://kubernetes.io/docs/tasks/debug/debug-application/determine-reason-pod-failure/#writing-and-reading-a-termination-message
termination_log() {
  echo "$*" > /dev/termination-log
}

__kubectl() {
  log "Run: kubectl $*"
  kubectl "$@"
}

__kubectl_ns() {
  __kubectl -n "$namespace" "$@"
}

get_daemon_job() {
  __kubectl_ns get daemonjobs.daemonjob.berquerant.github.io "$daemon_job_name" -o yaml
}

get_daemon_cronjob() {
  __kubectl_ns get daemoncronjobs.daemonjob.berquerant.github.io "$daemon_cronjob_name" -o yaml
}

workspace=""
# Save the original job template.
base_job_manifest=""
# Save the original spec.nodeSelector.
node_selector=""

# Kustomization patches.
patch=""
# Kustomization base manifest.
base=""

# Set up a kustomization directory.
setup() {
  workspace="$(mktemp -d)"
  patch="${workspace}/patch.yaml"
  base="${workspace}/base.yaml"
  cat <<EOS > "${workspace}/kustomization.yaml"
resources:
- base.yaml
patches:
- target:
    group: batch
    version: v1
    kind: Job
    name: ${base_job_name}
  path: patch.yaml
EOS

  base_job_manifest="$(mktemp)"
  node_selector="$(mktemp)"
  local __manifest
  __manifest="$(mktemp)"
  if [[ -n "$daemon_job_name" ]] ; then
    log "Get daemon job ${daemon_job_name} ..."
    if ! get_daemon_job > "$__manifest" ; then
      termination_log "Failed to get DaemonJob ${daemon_job_name}"
      return 1
    fi
    yq '.spec.nodeSelector // {}' "$__manifest" > "$node_selector"
    yq '.spec.jobTemplate' "$__manifest" > "$base_job_manifest"
    return
  fi

  log "Get daemon cronjob ${daemon_cronjob_name} ..."
  if ! get_daemon_cronjob > "$__manifest" ; then
    termination_log "Failed to get DaemonCronJob ${daemon_cronjob_name}"
    return 1
  fi
  yq '.spec.nodeSelector // {}' "$__manifest" > "$node_selector"
  yq '.spec.cronJobTemplate.spec.jobTemplate' "$__manifest" > "$base_job_manifest"
}

#
# Set up the workspace.
#
setup


indent() {
  awk -v n=$1 'BEGIN{for(i=1; i<n; i++) s=s " "}{print s $0}'
}

# Escape path of JSON 6902 patch.
escape_path() {
  sed 's|/|~1|g'
}

gen_list_nodes_options() {
  if [[ "$(yq '. // {} | length' "$node_selector")" -gt 0 ]] ; then
    echo -n "-l "
    yq 'to_entries | map(.key + "=" + .value) | join(",")' "$node_selector"
  fi
}

list_nodes() {
  if ! __kubectl get node -o=custom-columns='NAME:.metadata.name' --no-headers $(gen_list_nodes_options) ; then
    termination_log "Failed to list nodes"
    return 1
  fi
}

get_base_job_tolerations() {
  if [[ "$(yq '.spec.tolerations // [] | length' "$base_job_manifest")" -gt 0 ]] ; then
    yq 'spec.template.spec.tolerations' "$base_job_manifest"
  fi
}

# Generate the base job manifest of Kustomization.
gen_base() {
  cat <<EOS > "$base" # overwrite base
apiVersion: batch/v1
kind: Job
EOS
  # Set job name to be patched.
  # Set empty dict to metadata.labels if null.
  yq ".metadata.name = \"${base_job_name}\" | .metadata.labels = (.metadata.labels // {})" "$base_job_manifest" >> "$base"
}

gen_job_name() {
  local -r __node="$1"
  echo "${self_name}-${__node}"
}

# Generate the patches of Kustomization.
gen_patch() {
  local -r __name="$1"
  local -r __node="$2"
  cat <<EOS > "$patch" # overwrite patch
- op: replace
  path: /metadata/name
  value: ${__name}
- op: replace
  path: /spec/template/spec/affinity
  value:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchFields:
          - key: metadata.name
            operator: In
            values:
            - ${__node}
- op: replace
  path: /spec/template/spec/tolerations
  value:
    - effect: NoExecute
      key: node.kubernetes.io/not-ready
      operator: Exists
    - effect: NoExecute
      key: node.kubernetes.io/unreachable
      operator: Exists
    - effect: NoSchedule
      key: node.kubernetes.io/disk-pressure
      operator: Exists
    - effect: NoSchedule
      key: node.kubernetes.io/memory-pressure
      operator: Exists
    - effect: NoSchedule
      key: node.kubernetes.io/pid-pressure
      operator: Exists
    - effect: NoSchedule
      key: node.kubernetes.io/unschedulable
      operator: Exists
$(get_base_job_tolerations | indent 4)
- op: replace
  path: /metadata/ownerReferences
  value:
    - apiVersion: batch/v1
      blockOwnerDeletion: true
      controller: true
      kind: Job
      name: ${self_name}
      uid: ${controller_uid}
- op: add
  path: /metadata/labels/$(echo "$daemon_job_node_label" | escape_path)
  value: ${__node}
- op: add
  path: /metadata/labels/$(echo "$daemon_job_role_label" | escape_path)
  value: ${daemon_job_role_value}
EOS
  if [[ -n "$daemon_job_name" ]] ; then
    cat << EOS >> "$patch"
- op: add
  path: /metadata/labels/$(echo "$daemon_job_name_label" | escape_path)
  value: ${daemon_job_name}
EOS
  else
    cat << EOS >> "$patch"
- op: add
  path: /metadata/labels/$(echo "$daemon_cronjob_name_label" | escape_path)
  value: ${daemon_cronjob_name}
EOS
  fi
}

apply_job() {
  local -r __node="$1"
  local __name
  __name="$(gen_job_name "$__node")"
  if ! gen_patch "$__name" "$__node" ; then
    termination_log "Failed to generate patch"
    return 1
  fi
  log "Apply ${__name} for node ${__node}"
  if __kubectl_ns apply -k "$workspace" ; then
    log "Applied ${__name} for node ${__node} successfully."
    return
  else
    log "Failed to apply ${__name} for node ${__node}."
    termination_log "Failed to apply Job for node ${__node}"
    return 1
  fi
}

log "Generate base job ..."
if ! gen_base ; then
  termination_log "Failed to generate base Job"
  exit 1
fi

faillog() {
  log "Node selector:"
  cat >&2 "$node_selector"
  log "Base:"
  cat >&2 "$base"
  log "Patch:"
  cat >&2 "$patch"
}
trap faillog ERR

log "List nodes ..."
list_nodes
list_nodes | while read -r node ; do
  log "Process node ${node} ..."
  apply_job "$node"
done

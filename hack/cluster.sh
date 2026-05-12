#!/bin/bash

set -eo pipefail

readonly container_prefix="daemonjob-k0s-"
readonly controller_name="${container_prefix}controller"
readonly worker_name_prefix="${container_prefix}worker-"
readonly k0s_version="$K0S_VERSION"

usage() {
  local -r __name="${0##*/}"
  cat >&2 <<EOS
${__name} -- create k0s cluster in docker

${__name} start controller
  Start k0s controller node: ${controller_name}.
${__name} start worker WORKER_NUMBER
  Start k0s worker node: ${worker_name_prefix}WORKER_NUMBER.
${__name} start cluster
  Start one controller and one worker.
${__name} stop controller
  Stop k0s controller node: ${controller_name}.
${__name} stop worker WORKER_NUMBER
  Stop k0s worker node: ${worker_name_prefix}WORKER_NUMBER.
${__name} stop cluster
  Destroy all nodes.
${__name} kubectl ARGS...
  Run kubectl command.
${__name} exist
  Check if the cluster exists.
  The check succeeds if there is at least one controller and one worker.
${__name} kubeconfig
  Generate kubeconfig.
${__name} load IMAGE
  Load IMAGE into workers.
${__name} -h
${__name} --help
  Show this help.
EOS
}

log() {
  echo >&2 "${0##*/}: $*"
}

retry() {
  for c in $(seq 40) ; do
    if (( c > 1 )) ; then
      sleep 3
    fi
    if "$@" >/dev/null 2>&1 ; then
      return
    fi
    log "Retry[${c}] $*"
  done
  return 1
}

# Create k0s cluster in docker
# https://docs.k0sproject.io/stable/k0s-in-docker/#run-k0s-in-docker

delete() {
  docker rm --force "$@"
}

start() {
  docker run -d --rm "$@"
}

delete_cluster() {
  log "Delete cluster..."
  docker ps --filter="name=^${container_prefix}" --format='{{.Names}}' | while read -r cname ; do
    log "Delete ${cname} ..."
    delete "$cname"
  done
  log "Cluster is deleted!"
}

create_cluster() {
  log "Create cluster..."
  start_controller
  add_worker 0
}

exist_cluster() {
  if [[ "$(kubectl get node --no-headers | wc -l)" -gt 0 ]] ; then
    log "Cluster is found!"
  else
    if kubectl get node >/dev/null 2>&1 ; then
      log "Cluster is found but no workers!"
    else
      log "Cluster is not found!"
    fi
    return 1
  fi
}

# https://docs.k0sproject.io/head/k0s-in-docker/
start_controller() {
  log "Start controller..."
  start --name "$controller_name" --hostname "$controller_name" \
         --read-only \
         -v /var/lib/k0s \
         --tmpfs /run \
         --tmpfs /tmp \
         -p 6443:6443 \
         "docker.io/k0sproject/k0s:${k0s_version}" \
         k0s controller
  log "Wait controller..."
  retry kubectl get node
  log "Controller is started!"
  retry get_kubeconfig
  get_kubeconfig
}

stop_controller() {
  log "Stop controller..."
  delete "$controller_name" || log "Failed to stop controller or controller is not found"
  log "Controller is stopped!"
}

kubectl() {
  docker exec "$controller_name" k0s kubectl "$@"
}

get_kubeconfig() {
  if [[ -z "$KUBECONFIG" ]] ; then
    log "Failed to create kubeconfig: KUBECONFIG is not found!"
    return 1
  fi
  log "Create kubeconfig: ${KUBECONFIG}"
  local __tmp
  __tmp="$(mktemp)"
  docker exec "$controller_name" k0s kubeconfig admin > "$__tmp"
  sed -e "$(awk '/server:/ {print NR; exit}' ${__tmp})s|https://.*:6443|https://localhost:6443|" "$__tmp" > "$KUBECONFIG"
}

load_image() {
  local -r __image="$1"
  if [[ -z "$__image" ]] ; then
    log "Failed to load image: no image!"
    return 1
  fi
  kubectl get node -o=custom-columns='NAME:.metadata.name' --no-headers | while read -r node ; do
    log "Load ${__image} into ${node} ..."
    docker save "$__image" | docker exec -i "$node" k0s ctr images import -
  done
}

get_token() {
  log "Acquire a join token for the worker..."
  docker exec "$controller_name" k0s token create --role=worker
}

# https://docs.k0sproject.io/head/k0s-in-docker/
add_worker() {
  local -r __number="$1"
  if [[ -z "$__number" ]] ; then
    log "Cannot add worker: no worker number!"
    return 1
  fi
  log "Add worker ${__name} ..."
  local -r __name="${worker_name_prefix}${__number}"
  local __token
  retry get_token
  __token="$(get_token)"
  start --name "${__name}" --hostname "${__name}" \
         -v /var/lib/k0s \
         -v /var/log/pods \
         --tmpfs /run \
         --privileged \
         "docker.io/k0sproject/k0s:${k0s_version}" \
         k0s worker "$__token"
  log "Wait worker ${__name} to start ..."
  retry kubectl get node "$__name"
  log "Wait for worker ${__name} to be ready ..."
  kubectl wait --for=condition=Ready "node/${__name}" --timeout=120s
  log "Worker ${__name} is started!"
}

remove_worker() {
  local -r __number="$1"
  if [[ -z "$__number" ]] ; then
    log "Cannot add worker: no worker number!"
    return 1
  fi
  local -r __name="${worker_name_prefix}${__number}"
  log "Drain worker ${__name} ..."
  kubectl drain --ignore-daemonsets --delete-emptydir-data "$__name"
  log "Delete worker ${__name} from the cluster..."
  kubectl delete node "$__name"
  log "Stop worker node ${__name} ..."
  delete "$__name"
  log "Worker ${__name} is stopped!"
}


if [[ "$#" = 0 ]] ; then
  usage
  exit 1
fi
readonly cmd="$1"
readonly subcmd="$2"
errmsg=""
shift
case "$cmd" in
  "kubeconfig") get_kubeconfig "$@" ;;
  "load") load_image "$@" ;;
  "kubectl")
    kubectl "$@" ;;
  "exist")
    exist_cluster "$@" ;;
  "start")
    case "$subcmd" in
      "controller") start_controller ;;
      "worker")
        shift
        add_worker "$@"
        ;;
      "cluster") create_cluster ;;
      *) errmsg="Unknown start sub command: ${subcmd}" ;;
    esac
    ;;
  "stop")
    case "$subcmd" in
      "controller") stop_controller ;;
      "worker")
        shift
        remove_worker "$@"
        ;;
      "cluster")  delete_cluster ;;
      *) errmsg="Unknown stop sub command: ${subcmd}" ;;
    esac
    ;;
  "--help" | "-h") usage ;;
  *) errmsg="Unknown command: ${cmd}" ;;
esac

if [[ -n "$errmsg" ]] ; then
  log "$errmsg"
  exit 1
fi

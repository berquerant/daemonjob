#!/bin/bash

set -e

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

log() {
  echo >&2 "${0##*/}: $*"
}

shellcheck() {
  "${d}/tool.sh" shellcheck \
                 --shell=bash \
                 --severity=warning \
                 --exclude=SC2155,SC2090,SC2046,SC2089,SC2120 \
                 "$@"
}

cd "$topd" >/dev/null
git ls-files | grep -E ".sh$" | {
  result=0
  while read -r file ; do
    log "Check ${file}"
    if ! shellcheck "$file" ; then
      result=1
    fi
  done
  exit "$result"
}

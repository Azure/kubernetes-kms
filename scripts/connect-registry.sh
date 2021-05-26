#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if [ "${KIND_NETWORK}" != "bridge" ]; then
  # wait for the kind network to exist
  for i in $(seq 1 25); do
    if docker network ls | grep "${KIND_NETWORK}"; then
      break
    else
      sleep 1
    fi
  done
  containers=$(docker network inspect "${KIND_NETWORK}" -f "{{range .Containers}}{{.Name}} {{end}}")
  needs_connect="true"
  for c in $containers; do
    if [ "$c" = "${REGISTRY_NAME}" ]; then
      needs_connect="false"
    fi
  done
  if [ "${needs_connect}" = "true" ]; then
    echo "connecting ${KIND_NETWORK} network to ${REGISTRY_NAME}"
    docker network connect "${KIND_NETWORK}" "${REGISTRY_NAME}" || true
  fi
fi

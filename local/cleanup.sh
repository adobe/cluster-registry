#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


ROOT_DIR="$(cd "$(dirname "$0")/.."; pwd)"

# load local env vars
source $ROOT_DIR/local/.env.local

echo 'Tearing down local dev...'
kind delete cluster --name $KIND_CLUSTERNAME
docker rm -f ${CONTAINER_API} ${CONTAINER_CC} ${CONTAINER_DB} ${CONTAINER_SQS}
docker network rm ${NETWORK}
docker volume prune -f

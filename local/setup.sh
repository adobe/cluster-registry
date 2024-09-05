#!/usr/bin/env bash
# shellcheck enable=check-unassigned-uppercase

echoerr()  {
    echo "$@" 1>&2
}

die() {
    message="$1"
    shift
    exit_code="${1:-1}"
    echoerr "$message"
    exit "$exit_code"
}

container_exists() {
    local container_name container_id
    container_name="$1"
    [[ -z "$container_name" ]] && die 'container_exists was called without any args passed.'
    container_id="$(docker ps -a -q --filter name="$container_name")" || die "Failed to get ID for possibly extant container named $container_name"
    [[ -n "$container_id" ]]
}

container_running() {
    local container_name container_id
    container_name="$1"
    [[ -z "$container_name" ]] && die 'container_running was called without any args passed.'
    container_id="$(docker ps -q --filter name="$container_name")" || die "Failed to get ID for possibly running container named $container_name"
    [[ -n "$container_id" ]]
}

prerequisites=(aws docker kind)
missing_prerequisite=false
for prerequisite in "${prerequisites[@]}"; do
    if ! command -v "$prerequisite" > /dev/null; then
        echoerr "$prerequisite is not in your PATH. Is it installed?"
        missing_prerequisite=true
    fi
done
[[ "$missing_prerequisite" = true ]] && die 'Missing prerequisite commands. Cannot continue.'

docker info -f json > /dev/null || die 'Cannot talk to the docker daemon. Ensure docker is running and your user can access the socket.'

# by default run all components
RUN_APISERVER="${1:-1}"
RUN_CLIENT="${2:-1}"
RUN_SYNC_MANAGER="${3:-1}"

ROOT_DIR="$(cd "$(dirname "$0")/.."; pwd)"

echo 'Loading local environment variables...'
# shellcheck source=./.env.local
source "${ROOT_DIR}/local/.env.local" || die "Failed to source ${ROOT_DIR}/local/.env.local"

docker_network_id="$(docker network ls -q --filter name="${NETWORK}")"

if [[ -z "$docker_network_id" ]] > /dev/null 2>&1; then
    echo -e "Create docker network ${NETWORK} ..."
    docker network create "${NETWORK}" || die "Failed to create docker network ${NETWORK}"
fi

if ! container_exists "${CONTAINER_DB}" > /dev/null 2>&1; then
    echo 'Create a local dynamodb...'
    docker run -d \
        --name "${CONTAINER_DB}" \
        -v dynamodb:/home/dynamodblocal/data \
        -p 8000:8000 \
        --network "${NETWORK}" \
        "${IMAGE_DB}" -jar DynamoDBLocal.jar -inMemory -sharedDb || die "Failed to create $CONTAINER_DB container."
    until curl -s -o /dev/null http://localhost:8000; do
        echo "Waiting for dynamodb container to start up..."
        sleep 1
    done
elif container_exists "${CONTAINER_DB}" && ! container_running "${CONTAINER_DB}"; then
    docker start "${CONTAINER_DB}" || die "Failed to start stopped $CONTAINER_DB container."
    until curl -s -o /dev/null http://localhost:8000; do
        echo "Waiting for dynamodb container to start up..."
        sleep 1
    done
fi

echo 'Create dynamodb schema...'
aws dynamodb delete-table --region "${AWS_REGION}" --table-name "${DB_TABLE_NAME}" --endpoint-url "${DB_ENDPOINT}" > /dev/null 2>&1
aws dynamodb create-table --region "${AWS_REGION}" --cli-input-json file://"${ROOT_DIR}"/local/database/schema.json --endpoint-url "${DB_ENDPOINT}" > /dev/null || die "Failed to create DB schema."

echo 'Populate database with dummy data..'
go run "${ROOT_DIR}"/local/database/import.go --input-file "${ROOT_DIR}"/local/database/dummy-data.yaml || die "Failed to populate the database with dummy data."

if ! container_exists "${CONTAINER_SQS}"; then
    echo 'Run a local sqs...'
    docker run -d \
        --name "${CONTAINER_SQS}" \
        -v "${ROOT_DIR}"/local/sqs/sqs.conf:/opt/elasticmq.conf \
        -p 9324:9324 \
        -p 9325:9325 \
        --network "${NETWORK}" \
        "${IMAGE_SQS}" || die "Failed to create $CONTAINER_SQS container."
elif container_exists "${CONTAINER_SQS}" && ! container_running "${CONTAINER_SQS}"; then
    docker start "${CONTAINER_SQS}" || die "Failed to start stopped $CONTAINER_SQS container."
fi

if ! container_exists "${CONTAINER_REDIS}"; then
    echo 'Run a local redis...'
    docker run -d \
        --name ${CONTAINER_REDIS} \
        -p 6379:6379 \
        --network ${NETWORK} \
        "${IMAGE_REDIS}" || die "Failed to create $IMAGE_REDIS container."
elif container_exists "${CONTAINER_REDIS}" && ! container_running "${CONTAINER_REDIS}"; then
    docker start "${CONTAINER_REDIS}" || die "Failed to start stopped $CONTAINER_REDIS container."
fi

if ! container_exists "${CONTAINER_OIDC}"; then
    echo 'Run mocking oidc instance'
    docker run -d \
        --name "${CONTAINER_OIDC}" \
        -v "${ROOT_DIR}"/local/oidc:/usr/share/nginx/html \
        -v "${ROOT_DIR}"/local/oidc/default.conf:/etc/nginx/config.d/default.conf:ro \
        -p 80:80 \
        --network "${NETWORK}" \
        "${IMAGE_OIDC}" || die "Failed to create $CONTAINER_OIDC container."
elif container_exists "${CONTAINER_OIDC}" && ! container_running "${CONTAINER_OIDC}"; then
    docker start "${CONTAINER_OIDC}" || die "Failed to start stopped $CONTAINER_OIDC container."
fi

echo 'Creating a local k8s cluster...'
kind delete cluster --name="${KIND_CLUSTERNAME}" > /dev/null 2>&1
kind create cluster --name="${KIND_CLUSTERNAME}" --kubeconfig="${ROOT_DIR}"/kubeconfig \
            --image=kindest/node:"${KIND_NODE_VERSION}" --config="${ROOT_DIR}/local/kind/kind.yaml" || die "Failed to create $KIND_CLUSTERNAME kind cluster."

echo 'Testing local k8s cluster...'
export KUBECONFIG=${ROOT_DIR}/kubeconfig
docker network connect "${NETWORK}" "${KIND_CLUSTERNAME}-control-plane" || die "Failed to connect $NETWORK docker network to $KIND_CLUSTERNAME-control-plane network."
cp "${ROOT_DIR}/kubeconfig" "${ROOT_DIR}/kubeconfig_client" || die "Failed to copy our client kubeconfig."
chmod +r "${ROOT_DIR}/kubeconfig_client" || die "Failed to make ${ROOT_DIR}/kubeconfig_client readable."

perl -pi.bak -e "s/0.0.0.0/${KIND_CLUSTERNAME}-control-plane/g" "${ROOT_DIR}/kubeconfig_client" || die "Failed to modify the kubeconfig server address."
kubectl cluster-info --context kind-k8s-cluster-registry || die "Can't get cluster info. There may be an issue with the kind cluster or your kubeconfig."
kubectl create ns cluster-registry || die "Failed to create cluster-registry namespace."

echo 'Installing cluster-registry-client prerequisites...'
make manifests || die "Failed to run the manifests make target."
kubectl --kubeconfig="${ROOT_DIR}/kubeconfig" apply -f "${ROOT_DIR}"/config/crd/bases/ || die "Failed to apply k8s manifests at ${ROOT_DIR}/config/crd/bases/"

echo 'Building docker images'
make --always-make image TAG="${TAG}" || die "Failed to build images."

if [[ "${RUN_APISERVER}" == 1 ]]; then
    echo 'Running cluster-registry api'
    if container_exists "${CONTAINER_API}"; then
        container_running "${CONTAINER_API}" && { docker stop "$CONTAINER_API" || die "Failed to stop cluster-registry api container $CONTAINER_API"; }
        docker rm "${CONTAINER_API}" || die "Failed to remove cluster-registry api container $CONTAINER_API"
    fi
    docker run -d \
        --name "${CONTAINER_API}" \
        -p 8080:8080 \
        -e AWS_REGION \
        -e AWS_ACCESS_KEY_ID \
        -e AWS_SECRET_ACCESS_KEY \
        -e DB_AWS_REGION \
        -e DB_ENDPOINT=http://"${CONTAINER_DB}":8000 \
        -e DB_TABLE_NAME="${DB_TABLE_NAME}" \
        -e DB_INDEX_NAME="${DB_INDEX_NAME}" \
        -e OIDC_ISSUER_URL=http://"${CONTAINER_OIDC}" \
        -e OIDC_CLIENT_ID \
        -e SQS_AWS_REGION \
        -e SQS_ENDPOINT=http://"${CONTAINER_SQS}":9324 \
        -e SQS_QUEUE_NAME="${SQS_QUEUE_NAME}" \
        -e SQS_BATCH_SIZE \
        -e SQS_WAIT_SECONDS \
        -e SQS_RUN_INTERVAL \
        -e API_RATE_LIMITER="${API_RATE_LIMITER}" \
        -e LOG_LEVEL="${LOG_LEVEL}" \
        -e API_HOST="${API_HOST}" \
        -e K8S_RESOURCE_ID="${K8S_RESOURCE_ID}" \
        -e API_TENANT_ID="${API_TENANT_ID}" \
        -e API_CLIENT_ID="${API_CLIENT_ID}" \
        -e API_CLIENT_SECRET="${API_CLIENT_SECRET}" \
        -e API_AUTHORIZED_GROUP_ID="${API_AUTHORIZED_GROUP_ID}" \
        -e API_CACHE_TTL \
        -e API_CACHE_REDIS_HOST=${CONTAINER_REDIS}:6379 \
        --network "${NETWORK}" \
        "${IMAGE_APISERVER}":"${TAG}" || die "Failed to create $CONTAINER_API container."
fi

if [[ "${RUN_CLIENT}" == 1 ]]; then
    echo 'Running cluster-registry-client'
    if container_exists "${CONTAINER_CLIENT}"; then
        container_running "${CONTAINER_CLIENT}" && { docker stop "$CONTAINER_CLIENT" || die "Failed to stop cluster-registry-client container $CONTAINER_CLIENT"; }
        docker rm "${CONTAINER_CLIENT}" || die "Failed to remove cluster-registry-client container $CONTAINER_CLIENT"
    fi
    docker run -d \
        --name "${CONTAINER_CLIENT}" \
        -v "${ROOT_DIR}/kubeconfig_client":/kubeconfig \
        -e AWS_ACCESS_KEY_ID \
        -e AWS_SECRET_ACCESS_KEY \
        -e KUBECONFIG=/kubeconfig \
        -e SQS_AWS_REGION \
        -e SQS_ENDPOINT=http://"${CONTAINER_SQS}":9324 \
        -e SQS_QUEUE_NAME="${SQS_QUEUE_NAME}" \
        --network "${NETWORK}" \
        "${IMAGE_CLIENT}":"${TAG}" || die "Failed to create $CONTAINER_CLIENT container."
fi

if [[ "${RUN_SYNC_MANAGER}" == 1 ]]; then
    echo 'Running cluster-registry-sync-manager'
    if container_exists "${CONTAINER_SYNC_MANAGER}"; then
        container_running "${CONTAINER_SYNC_MANAGER}" && { docker stop "CONTAINER_SYNC_MANAGER" || die "Failed to stop cluster-registry-sync-manager container $CONTAINER_SYNC_MANAGER"; }
        docker rm "${CONTAINER_SYNC_MANAGER}" || die "Failed to remove cluster-registry-sync-manager container CONTAINER_SYNC_MANAGER"
    fi
    docker run -d \
        --name "${CONTAINER_SYNC_MANAGER}" \
        -v "${ROOT_DIR}/kubeconfig_client":/kubeconfig \
        -e AWS_ACCESS_KEY_ID \
        -e AWS_SECRET_ACCESS_KEY \
        -e KUBECONFIG=/kubeconfig \
        -e SQS_AWS_REGION \
        -e SQS_ENDPOINT=http://"${CONTAINER_SQS}":9324 \
        -e SQS_QUEUE_NAME="${SQS_QUEUE_NAME}" \
        --network "${NETWORK}" \
        "${IMAGE_SYNC_MANAGER}":"${TAG}" || die "Failed to create $CONTAINER_SYNC_MANAGER container."
fi

echo 'Local stack was set up successfully.'

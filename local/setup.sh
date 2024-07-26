#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# by default run the apiserver and client
RUN_APISERVER="${1:-1}"
RUN_CLIENT="${2:-1}"

ROOT_DIR="$(cd "$(dirname "$0")/.."; pwd)"

echo 'Loading local environment variables...'
source ${ROOT_DIR}/local/.env.local

echo -e "Create docker network ${NETWORK} ..."
if [[ $(docker network ls | grep ${NETWORK}) == "" ]]; then
    docker network create ${NETWORK}
fi

echo 'Run a local dynamodb...'
docker run -d \
    --name ${CONTAINER_DB} \
	-v dynamodb:/home/dynamodblocal/data \
    -p 8000:8000 \
	--network ${NETWORK} \
    ${IMAGE_DB} -jar DynamoDBLocal.jar -inMemory -sharedDb

echo 'Sleeping 3s in order to let dynamodb container to start up'
sleep 3

echo 'Create dynamodb schema...'
aws dynamodb delete-table --table-name ${DB_TABLE_NAME} --endpoint-url $DB_ENDPOINT > /dev/null 2>&1 || true
aws dynamodb create-table --cli-input-json file://${ROOT_DIR}/local/database/schema.json --endpoint-url $DB_ENDPOINT > /dev/null

echo 'Populate database with dummy data..'
go run ${ROOT_DIR}/local/database/import.go --input-file ${ROOT_DIR}/local/database/dummy-data.yaml

echo 'Run a local sqs...'
docker run -d \
    --name ${CONTAINER_SQS} \
	-v ${ROOT_DIR}/local/sqs/sqs.conf:/opt/elasticmq.conf \
    -p 9324:9324 \
    -p 9325:9325 \
	--network ${NETWORK} \
    ${IMAGE_SQS}

echo 'Run a local redis...'
docker run -d \
    --name ${CONTAINER_REDIS} \
    -p 6379:6379 \
	--network ${NETWORK} \
    ${IMAGE_REDIS}

echo 'Run mocking oidc instance'
docker run -d \
    --name ${CONTAINER_OIDC} \
	-v ${ROOT_DIR}/local/oidc:/usr/share/nginx/html \
	-v ${ROOT_DIR}/local/oidc/default.conf:/etc/nginx/config.d/default.conf:ro \
    -p 80:80 \
	--network ${NETWORK} \
    ${IMAGE_OIDC}

echo 'Creating a local k8s cluster...'
kind delete cluster --name="${KIND_CLUSTERNAME}" > /dev/null 2>&1 || true
kind create cluster --name="${KIND_CLUSTERNAME}" --kubeconfig=${ROOT_DIR}/kubeconfig \
			--image=kindest/node:"${KIND_NODE_VERSION}" --config="${ROOT_DIR}/local/kind/kind.yaml"

echo 'Testing local k8s cluster...'
export KUBECONFIG=${ROOT_DIR}/kubeconfig
docker network connect "${NETWORK}" "${KIND_CLUSTERNAME}-control-plane"
cp "${ROOT_DIR}/kubeconfig" "${ROOT_DIR}/kubeconfig_client"
chmod +r "${ROOT_DIR}/kubeconfig_client"

perl -pi.bak -e "s/0.0.0.0/${KIND_CLUSTERNAME}-control-plane/g" "${ROOT_DIR}/kubeconfig_client"
kubectl cluster-info --context kind-k8s-cluster-registry
kubectl create ns cluster-registry

echo 'Installing cluster-registry-client prerequisites...'
make manifests
kubectl --kubeconfig="${ROOT_DIR}/kubeconfig" apply -f ${ROOT_DIR}/config/crd/bases/

echo 'Building docker images'
make --always-make image TAG="${TAG}"

if [[ "${RUN_APISERVER}" == 1 ]]; then
	echo 'Running cluster-registry api'
	docker run -d \
		--name ${CONTAINER_API} \
		-p 8080:8080 \
		-e AWS_REGION \
		-e AWS_ACCESS_KEY_ID \
		-e AWS_SECRET_ACCESS_KEY \
		-e DB_AWS_REGION \
		-e DB_ENDPOINT=http://${CONTAINER_DB}:8000 \
		-e DB_TABLE_NAME=${DB_TABLE_NAME} \
		-e DB_INDEX_NAME=${DB_INDEX_NAME} \
		-e OIDC_ISSUER_URL=http://${CONTAINER_OIDC} \
		-e OIDC_CLIENT_ID \
		-e SQS_AWS_REGION \
		-e SQS_ENDPOINT=http://${CONTAINER_SQS}:9324 \
		-e SQS_QUEUE_NAME=${SQS_QUEUE_NAME} \
		-e API_RATE_LIMITER=${API_RATE_LIMITER} \
		-e LOG_LEVEL=${LOG_LEVEL} \
		-e API_HOST=${API_HOST} \
		-e K8S_RESOURCE_ID=${K8S_RESOURCE_ID} \
		-e API_TENANT_ID=${API_TENANT_ID} \
		-e API_CLIENT_ID=${API_CLIENT_ID} \
		-e API_CLIENT_SECRET=${API_CLIENT_SECRET} \
		-e API_AUTHORIZED_GROUP_ID=${API_AUTHORIZED_GROUP_ID} \
		--network ${NETWORK} \
		${IMAGE_APISERVER}:${TAG}
fi

if [[ "${RUN_CLIENT}" == 1 ]]; then
	echo 'Running cluster-registry-client'
	docker run -d \
		--name ${CONTAINER_CLIENT} \
		-v "${ROOT_DIR}/kubeconfig_client":/kubeconfig \
		-e AWS_ACCESS_KEY_ID \
		-e AWS_SECRET_ACCESS_KEY \
		-e KUBECONFIG=/kubeconfig \
		-e SQS_AWS_REGION \
		-e SQS_ENDPOINT=http://${CONTAINER_SQS}:9324 \
		-e SQS_QUEUE_NAME=${SQS_QUEUE_NAME} \
		--network ${NETWORK} \
		${IMAGE_CLIENT}:${TAG}
fi

echo 'Local stack was set up successfully.'

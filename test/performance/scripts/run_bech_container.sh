#!/usr/bin/env bash

# This script is so we can pass the enviorment variables from the /local/.env.local
# file to the container that runs the benchmark.sh script when ran from the Makefile

if [ -z "${TAG}" ]; then
    TAG="v$(cat "$(git rev-parse --show-toplevel)/VERSION")-$(git rev-parse --short HEAD)"
fi

if [ -z "${IMAGE_PERFORMANCE_TESTS}" ]; then
    IMAGE_PERFORMANCE_TESTS="ghcr.io/adobe/performance-tests"
fi

if [ -z "${NETWORK}" ]; then
    NETWORK="cluster-registry-net"
fi

docker run --rm --network="${NETWORK}" \
           -e APISERVER_ENDPOINT \
           -e PERFORMANCE_TEST_TIME \
           -e APISERVER_AUTH_TOKEN \
           "${IMAGE_PERFORMANCE_TESTS}:${TAG}"
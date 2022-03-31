#!/usr/bin/env bash

# This script is so we can pass the enviorment variables from the /local/.env.local
# file to the container that runs the benchmark.sh script when ran from the Makefile.
#
# The env variables with the prefix LOCAL_ENV are from the env of the user that ran
# the make target, and the ones without the prefix are from the /local/.env.local file.
#
# Only the container run env variables have defaults in this script the APISERVER_ENDPOINT
# and PERFORMANCE_TEST_TIME have defaults inside the container. The APISERVER_AUTH_TOKEN
# is required.

# If LOCAL_ENV_ exists and is not empty
if [[ -n "${LOCAL_ENV_APISERVER_ENDPOINT}" ]]; then
    APISERVER_ENDPOINT="${LOCAL_ENV_APISERVER_ENDPOINT}"
fi

if [[ -n "${LOCAL_ENV_APISERVER_AUTH_TOKEN}" ]]; then
    APISERVER_AUTH_TOKEN="${LOCAL_ENV_APISERVER_AUTH_TOKEN}"
fi

if [[ -n "${LOCAL_ENV_PERFORMANCE_TEST_TIME}" ]]; then
    PERFORMANCE_TEST_TIME="${LOCAL_ENV_PERFORMANCE_TEST_TIME}"
fi

if [[ -n "${LOCAL_ENV_NETWORK}" ]]; then
    NETWORK="${LOCAL_ENV_NETWORK}"
elif [ -z "${NETWORK}" ]; then
    NETWORK="cluster-registry-net"
fi

if [[ -n "${LOCAL_ENV_IMAGE_PERFORMANCE_TESTS}" ]]; then
    IMAGE_PERFORMANCE_TESTS="${LOCAL_ENV_IMAGE_PERFORMANCE_TESTS}"
elif [ -z "${IMAGE_PERFORMANCE_TESTS}" ]; then
    IMAGE_PERFORMANCE_TESTS="ghcr.io/adobe/performance-tests"
fi

if [[ -n "${LOCAL_ENV_TAG}" ]]; then
    TAG="${LOCAL_ENV_TAG}"
elif [ -z "${TAG}" ]; then
    TAG="v$(cat "$(git rev-parse --show-toplevel)/VERSION")-$(git rev-parse --short HEAD)"
fi

docker run --rm --network="${NETWORK}" \
           -e APISERVER_ENDPOINT \
           -e PERFORMANCE_TEST_TIME \
           -e APISERVER_AUTH_TOKEN \
           "${IMAGE_PERFORMANCE_TESTS}:${TAG}"

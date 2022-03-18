#!/usr/bin/env bash


set -e # exit immediately when a command fails
set -o pipefail # only exit with zero if all commands of the pipeline exit successfully

REGISTRY="ghcr.io"
REPOSITORY="adobe"

# If you move this file rethink the ROOT_DIR var
ROOT_DIR="$(cd "$(dirname "$0")/../../.."; pwd)"

IMAGE_SUFFIX="-dev"
if [[ "$TAG" =~ ^v[0-9]+\.[0-9]+ ]] || [ "${TAG}" == "main" ]; then
	IMAGE_SUFFIX=""
else
	TAG="v$(cat "$(git rev-parse --show-toplevel)/VERSION")-$(git rev-parse --short HEAD)"
fi

# The default will not be assinged if the var is empty only if it does not exist
default_image_name="${REGISTRY}/${REPOSITORY}/performance-tests"
IMAGE_PERFORMANCE_TESTS="${IMAGE_PERFORMANCE_TESTS:-"${default_image_name}"}"
IMAGE_PERFORMANCE_TESTS="${IMAGE_PERFORMANCE_TESTS}${IMAGE_SUFFIX}"


printf "Realeasing image %s...\n\n" "${IMAGE_PERFORMANCE_TESTS}:${TAG}"

make -C "${ROOT_DIR}" --always-make build-performance \
    TAG="${TAG}" \
    IMAGE_PERFORMANCE_TESTS="${IMAGE_PERFORMANCE_TESTS}"

docker push "${IMAGE_PERFORMANCE_TESTS}:${TAG}"

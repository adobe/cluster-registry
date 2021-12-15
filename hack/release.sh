#!/usr/bin/env bash
# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail

REGISTRY="ghcr.io"

export IMAGE_API="${IMAGE_API:-"adobe/cluster-registry-api"}"
export IMAGE_CC="${IMAGE_CC:-"adobe/cluster-registry-client"}"
export TAG="${GITHUB_REF##*/}"

echo "tag: ${TAG}"

IMAGE_SUFFIX="-dev"

if [[ "$TAG" =~ ^v[0-9]+\.[0-9]+ ]] || [ "${TAG}" == "main" ]; then
	IMAGE_SUFFIX=""
else
	TAG="v$(cat "$(git rev-parse --show-toplevel)/VERSION")-$(git rev-parse --short HEAD)"
fi

API="${REGISTRY}/${IMAGE_API}${IMAGE_SUFFIX}"
CC="${REGISTRY}/${IMAGE_CC}${IMAGE_SUFFIX}"

for img in ${API} ${CC}; do
	echo "Building image: $img:$TAG"
done

make --always-make image TAG="${TAG}"

docker tag "${IMAGE_API}:${TAG}" "${API}:${TAG}"
docker tag "${IMAGE_CC}:${TAG}" "${CC}:${TAG}"

docker push "${API}:${TAG}"
docker push "${CC}:${TAG}"
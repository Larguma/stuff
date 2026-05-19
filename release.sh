#!/bin/bash

set -e

echo -n "Did you update the version? (./release.sh v1.0.0)"
read

TAG=${1:-latest}
IMAGE_NAME="larguma/stuff"

echo "Building image $IMAGE_NAME:$TAG..."
docker build --build-arg "VERSION=$TAG" -t "$IMAGE_NAME:$TAG" .

echo "Pushing image $IMAGE_NAME:$TAG to Docker Hub..."
docker push "$IMAGE_NAME:$TAG"

if [ "$TAG" != "latest" ]; then
    echo "Tagging and pushing $IMAGE_NAME:latest..."
    docker tag "$IMAGE_NAME:$TAG" "$IMAGE_NAME:latest"
    docker push "$IMAGE_NAME:latest"
fi

echo "Done!"

#!/bin/bash
set -eux
image="$1"
tag="$2"

# build
docker build -t $image:$tag ./
docker tag $image:$tag $image:latest
# push
docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD";
docker push $image:$tag
docker push $image:latest

# build binaries
$GOPATH/bin/gox -os="linux darwin windows" -arch="amd64" github.com/rerorero/meshem/src/meshem github.com/rerorero/meshem/src/meshemctl

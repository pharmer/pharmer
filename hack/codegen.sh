#!/bin/bash

set -x

GOPATH=$(go env GOPATH)
PACKAGE_NAME=pharmer.dev/pharmer
REPO_ROOT="$GOPATH/src/$PACKAGE_NAME"
DOCKER_REPO_ROOT="/go/src/$PACKAGE_NAME"

pushd $REPO_ROOT

mkdir -p "$REPO_ROOT"/api/api-rules

# Generate deep copies
docker run --rm -ti -u $(id -u):$(id -g) \
  -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  appscode/gengo:release-1.14 deepcopy-gen \
  --go-header-file "hack/gengo/boilerplate.go.txt" \
  --input-dirs "$PACKAGE_NAME/apis/v1alpha1" \
  --output-file-base zz_generated.deepcopy

# # Generate protobuf definitions
# docker run --rm -ti -u $(id -u):$(id -g) \
#   -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
#   -w "$DOCKER_REPO_ROOT" \
#   appscode/protoc:release-1.13 go-to-protobuf \
#   --go-header-file "hack/gengo/boilerplate.go.txt" \
#   --proto-import=/go/src/pharmer.dev/pharmer/vendor \
#   --packages=-k8s.io/api/core/v1,pharmer.dev/pharmer/apis/v1alpha1 \
#   --apimachinery-packages=-k8s.io/apimachinery/pkg/apis/meta/v1

popd

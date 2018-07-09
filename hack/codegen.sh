#!/bin/bash

set -x

GOPATH=$(go env GOPATH)
PACKAGE_NAME=github.com/pharmer/pharmer
REPO_ROOT="$GOPATH/src/$PACKAGE_NAME"
DOCKER_REPO_ROOT="/go/src/$PACKAGE_NAME"

pushd $REPO_ROOT

# Generate deep copies
docker run --rm -ti -u $(id -u):$(id -g) \
  -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  appscode/gengo:release-1.11 deepcopy-gen \
  --v 1 --logtostderr \
  --go-header-file "hack/gengo/boilerplate.go.txt" \
  --input-dirs "$PACKAGE_NAME/apis/v1alpha1" \
  --output-file-base zz_generated.deepcopy

# Generate protobuf definitions
docker run --rm -ti -u $(id -u):$(id -g) \
  -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  appscode/protoc:release-1.11 go-to-protobuf \
  --go-header-file "hack/gengo/boilerplate.go.txt" \
  --proto-import=/go/src/github.com/pharmer/pharmer/vendor \
  --packages=-k8s.io/api/core/v1,github.com/pharmer/pharmer/apis/v1alpha1 \
  --apimachinery-packages=-k8s.io/apimachinery/pkg/apis/meta/v1

popd

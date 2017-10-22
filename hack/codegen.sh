#!/bin/bash

set -x

GOPATH=$(go env GOPATH)
PACKAGE_NAME=github.com/appscode/pharmer
REPO_ROOT="$GOPATH/src/$PACKAGE_NAME"
DOCKER_REPO_ROOT="/go/src/$PACKAGE_NAME"

pushd $REPO_ROOT

# Generate deep copies
docker run --rm -ti -u $(id -u):$(id -g) \
    -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
    -w "$DOCKER_REPO_ROOT" \
    appscode/gengo:release-1.8 deepcopy-gen \
    --v 1 --logtostderr \
    --go-header-file "hack/gengo/boilerplate.go.txt" \
    --input-dirs "$PACKAGE_NAME/apis/v1alpha1" \
    --output-file-base zz_generated.deepcopy

# Generate deep copies
docker run --rm -ti -u $(id -u):$(id -g) \
    -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
    -w "$DOCKER_REPO_ROOT" \
    appscode/protoc:release-1.8 go-to-protobuf \
    --go-header-file "hack/gengo/boilerplate.go.txt" \
    --proto-import=/go/src/github.com/appscode/pharmer/vendor \
    --packages=-k8s.io/api/core/v1,github.com/appscode/pharmer/apis/v1alpha1 \
    --apimachinery-packages=-k8s.io/apimachinery/pkg/apis/meta/v1

popd

# go-to-protobuf \
#   --v 3 --logtostderr \
#   --go-header-file hack/gengo/boilerplate.go.txt \
#   --proto-import=/home/tamal/go/src/github.com/appscode/pharmer/vendor \
#   --packages=-k8s.io/api/core/v1,github.com/appscode/pharmer/apis/v1alpha1 \
#   --apimachinery-packages=-k8s.io/apimachinery/pkg/apis/meta/v1

#!/bin/bash

set -eou pipefail

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/pharmer/pharmer

source "$REPO_ROOT/hack/libbuild/common/public_image.sh"

IMG=inspector-nginx
TAG=alpine

build() {
	pushd $(dirname "${BASH_SOURCE}")
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd
	popd
}

binary_repo $@

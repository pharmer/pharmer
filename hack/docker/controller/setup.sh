#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
ROOT=$GOPATH
REPO_ROOT=$GOPATH/src/github.com/pharmer/pharmer

source "$REPO_ROOT/hack/libbuild/common/pharmer_image.sh"

APPSCODE_ENV=${APPSCODE_ENV:-dev}
IMG=machine-controller

DIST=$GOPATH/src/github.com/pharmer/pharmer/dist
mkdir -p $DIST
if [ -f "$DIST/.tag" ]; then
	export $(cat $DIST/.tag | xargs)
fi

clean() {
    pushd $GOPATH/src/github.com/pharmer/pharmer/hack/docker/controller
    rm pharmer Dockerfile
    popd
}

build_binary() {
    pushd $GOPATH/src/github.com/pharmer/pharmer
    ./hack/builddeps.sh
    ./hack/make.py build
    detect_tag $DIST/.tag
    popd
}

build_docker() {
    pushd $GOPATH/src/github.com/pharmer/pharmer/hack/docker/controller
    cp $DIST/pharmer/pharmer-linux-amd64 machine-controller
    chmod 755 machine-controller

    cat >Dockerfile <<EOL
FROM ubuntu:latest as kubeadm
RUN apt-get update
RUN apt-get install -y curl
RUN curl -fsSL https://dl.k8s.io/release/v1.13.2/bin/linux/amd64/kubeadm > /usr/bin/kubeadm
RUN chmod a+rx /usr/bin/kubeadm

FROM ubuntu:latest
WORKDIR /

RUN set -x \
   && apt update \
  && apt install  -y curl

COPY machine-controller /usr/bin/machine-controller
COPY --from=kubeadm /usr/bin/kubeadm /usr/bin/kubeadm

ENTRYPOINT ["machine-controller"]
EOL
    local cmd="docker build -t pharmer/$IMG:$TAG ."
    echo $cmd; $cmd

    rm machine-controller Dockerfile
    popd
}

build() {
    build_binary
    build_docker
}

docker_push() {
    if [ "$APPSCODE_ENV" = "prod" ]; then
        echo "Nothing to do in prod env. Are you trying to 'release' binaries to prod?"
        exit 0
    fi
    if [ "$TAG_STRATEGY" = "git_tag" ]; then
        echo "Are you trying to 'release' binaries to prod?"
        exit 1
    fi
    hub_canary
}

docker_release() {
    if [ "$APPSCODE_ENV" != "prod" ]; then
        echo "'release' only works in PROD env."
        exit 1
    fi
    if [ "$TAG_STRATEGY" != "git_tag" ]; then
        echo "'apply_tag' to release binaries and/or docker images."
        exit 1
    fi
    hub_up
}

source_repo $@

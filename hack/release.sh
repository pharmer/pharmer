#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

export APPSCODE_ENV=prod

pushd "$(go env GOPATH)/src/github.com/appscode/pharmer"
rm -rf dist
./hack/make.py build
./hack/make.py push
./hack/make.py update_registry
popd

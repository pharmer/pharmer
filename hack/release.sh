#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

pushd "$(go env GOPATH)/src/github.com/appscode/pharmer"
rm -rf dist
APPSCODE_ENV=prod ./hack/make.py build
APPSCODE_ENV=prod ./hack/make.py push
./hack/make.py update_registry
popd

#!/bin/bash

set -x -e

#install python pip
apt-get update >/dev/null
apt-get install -y python python-pip >/dev/null

#copy pharmer to $GOPATH
mkdir -p $GOPATH/src/github.com/pharmer
cp -r pharmer $GOPATH/src/github.com/pharmer
pushd $GOPATH/src/pharmer.dev/pharmer

#build pharmer
./hack/builddeps.sh
./hack/make.py

popd

#copy pharmer here for uploading to s3
cp $GOPATH/bin/pharmer .

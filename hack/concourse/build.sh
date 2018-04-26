#!/bin/bash

set -x -e

#install python pip
apt-get update > /dev/null
apt-get install -y python python-pip > /dev/null

#install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
 chmod +x ./kubectl
 mv ./kubectl /bin/kubectl

#copy pharmer to $GOPATH
mkdir -p $GOPATH/src/github.com/pharmer
cp -r pharmer $GOPATH/src/github.com/pharmer
pushd $GOPATH/src/github.com/pharmer/pharmer

#build pharmer
./hack/builddeps.sh
./hack/make.py

popd

#copy pharmer here for uploading to s3
cp $GOPATH/bin/pharmer .

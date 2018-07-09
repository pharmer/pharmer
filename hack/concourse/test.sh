#!/bin/bash

set -x -e

#install python pip
apt-get update >/dev/null
apt-get install -y python python-pip >/dev/null

#install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &>/dev/null
chmod +x ./kubectl
mv ./kubectl /bin/kubectl

#copy pharmer to $GOPATH
mkdir -p $GOPATH/src/github.com/pharmer
cp -r pharmer $GOPATH/src/github.com/pharmer
pushd $GOPATH/src/github.com/pharmer/pharmer

#if test_only == false, we need to build pharmer, otherwise, get pharmer from s3
if [ -z "$test_only" ]; then
  #build pharmer
  ./hack/builddeps.sh
  ./hack/make.py
  popd

  if [ -n "$build_only" ]; then
    cp $GOPATH/bin/pharmer pharmer-bin/pharmer
    exit 0
  fi
else
  popd
  mkdir -p $GOPATH/bin
  cp pharmer-bin/pharmer $GOPATH/bin/pharmer
fi

chmod +x $GOPATH/bin/pharmer

pushd $GOPATH/src/github.com/pharmer/pharmer
#name of the cluster
NAME=pharmer-$(git rev-parse --short HEAD)
popd

#delete cluster after running tests
function cleanup() {
  pharmer get cluster
  pharmer delete cluster $NAME
  pharmer get cluster
  sleep 300
  pharmer apply $NAME
  pharmer get cluster
}
trap cleanup EXIT

#create k8s cluster
cp creds/creds/$CRED.json cred.json

pharmer create credential --from-file=cred.json --provider=$CredProvider cred
pharmer create cluster $NAME --provider=$ClusterProvider --zone=$ZONE --nodes=$NODE=1 --credential-uid=cred --kubernetes-version=v1.9.0
pharmer apply $NAME
pharmer use cluster $NAME
sleep 300 #make sure that all the nodes are ready
kubectl get nodes

#https://github.com/pharmer/k8s-conformance/tree/master/v1.9/appscode
curl -L https://raw.githubusercontent.com/cncf/k8s-conformance/master/sonobuoy-conformance.yaml | kubectl apply -f -
sleep 900
nohup kubectl logs -f -n sonobuoy sonobuoy &

while [ $(
  grep -q "no-exit was specified, sonobuoy is now blocking" nohup.out
  echo $?
) == 1 ]; do
  sleep 300
done

kubectl cp sonobuoy/sonobuoy:/tmp/sonobuoy ./results --namespace=sonobuoy
tar xfz results/*.tar.gz

pushd plugins/e2e/results
cat e2e.log

if [ "$(tail -1 e2e.log)" == "Test Suite Failed" ]; then
  exit 1
fi
popd

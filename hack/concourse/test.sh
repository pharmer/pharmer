#!/bin/bash

set -x -e

#install python pip
apt-get update > /dev/null
apt-get install -y python python-pip > /dev/null

#copy pharmer to $GOPATH
mkdir -p $GOPATH/src/github.com/pharmer
cp -r pharmer $GOPATH/src/github.com/pharmer
pushd $GOPATH/src/github.com/pharmer/pharmer

#build
./hack/builddeps.sh
./hack/make.py

NAME=pharmer-$(git rev-parse HEAD) #name of the cluster
popd

function cleanup {
    pharmer get cluster
    pharmer delete cluster $NAME
    pharmer get cluster
    sleep 120
    pharmer apply $NAME
    pharmer get cluster
}
trap cleanup EXIT

cat > cred.json <<EOF
{
        "token" : "$TOKEN"
}
EOF

pharmer create credential --from-file=cred.json --provider=$CredProvider cred
pharmer create cluster $NAME --provider=$ClusterProvider --zone=nyc3 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.9.0
pharmer apply $NAME
pharmer use cluster $NAME
kubectl get nodes


curl -L https://raw.githubusercontent.com/cncf/k8s-conformance/master/sonobuoy-conformance.yaml | kubectl apply -f -
nohup kubectl logs -f -n sonobuoy sonobuoy &

while [ $(grep -q "no-exit was specified, sonobuoy is now blocking" nohup.out; echo $?) == 1 ]
do
    sleep 300
done

pushd results/plugins/e2e/results
cat e2e.log

if [ "$(tail -1 e2e.log)" == "Test Suite Failed" ]; then
    exit 1
fi
popd

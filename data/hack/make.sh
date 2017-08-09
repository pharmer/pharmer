#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

RETVAL=0

deps() {
    echo '"Happiness belongs to the self sufficient." - Aristotle'
}

build() {
    local -r owd=$PWD
    cd $GOPATH/src/github.com/appscode/data
    go get github.com/jteeuwen/go-bindata/...
    go-bindata -ignore=\\.go -mode=0644 -modtime=1453795200 -o files/data.go -pkg files files/...
    goimports -w *.go
    go build ./...
    cd $owd
}

if [ $# -eq 0 ]; then
	build
	exit $RETVAL
fi

case "$1" in
	deps)
		deps
		;;
	build)
		build
		;;
	*)	(10)
		echo $"Usage: $0 {deps|build}"
		RETVAL=1
esac
exit $RETVAL

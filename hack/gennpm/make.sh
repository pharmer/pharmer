#!/usr/bin/env bash

pushd $GOPATH/src/github.com/pharmer/pharmer/hack/gennpm
go run main.go
popd

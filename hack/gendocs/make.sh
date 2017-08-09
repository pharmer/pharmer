#!/usr/bin/env bash

pushd $GOPATH/src/github.com/appscode/pharmer/hack/gendocs
go run main.go
popd

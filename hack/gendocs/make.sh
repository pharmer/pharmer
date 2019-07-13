#!/usr/bin/env bash

pushd $GOPATH/src/pharmer.dev/pharmer/hack/gendocs
go run main.go provider.go
popd

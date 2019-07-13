package main

import (
	"os"

	_ "github.com/gogo/protobuf/gogoproto"
	_ "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"kmodules.xyz/client-go/logs"
	_ "pharmer.dev/pharmer/cloud/providers"
	"pharmer.dev/pharmer/cmds"
	_ "pharmer.dev/pharmer/store/providers"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewRootCmd(os.Stdin, os.Stdout, os.Stderr, Version).Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

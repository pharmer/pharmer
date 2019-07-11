package main

import (
	"os"

	_ "github.com/gogo/protobuf/gogoproto"
	_ "github.com/pharmer/pharmer/cloud/providers"
	"github.com/pharmer/pharmer/cmds"
	_ "github.com/pharmer/pharmer/store/providers"
	_ "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"kmodules.xyz/client-go/logs"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewRootCmd(os.Stdin, os.Stdout, os.Stderr, Version).Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

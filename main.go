package main

import (
	"os"

	logs "github.com/appscode/log/golog"
	_ "github.com/appscode/pharmer/cloud/providers"
	"github.com/appscode/pharmer/cmds"
	_ "github.com/appscode/pharmer/store/providers"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewRootCmd(os.Stdin, os.Stdout, os.Stderr, Version).Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

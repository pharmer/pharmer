package main

import (
	"os"

	logs "github.com/appscode/log/golog"
	_ "github.com/appscode/pharmer/cloud/providers"
	"github.com/appscode/pharmer/cmds"
	_ "github.com/appscode/pharmer/storage/providers"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewRootCmd(Version).Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

package main

import (
	"os"

	logs "github.com/appscode/go/log/golog"
	"github.com/pharmer/pharmer/hack/gendata/cmds"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmds.NewCmdLoadData().Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

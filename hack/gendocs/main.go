package main

import (
	"fmt"
	"log"
	"os"

	"github.com/appscode/go/runtime"
	"github.com/appscode/pharmer/cmds"
	"github.com/spf13/cobra/doc"
)

// ref: https://github.com/spf13/cobra/blob/master/doc/md_docs.md
func main() {
	rootCmd := cmds.NewRootCmd("")
	dir := runtime.GOPath() + "/src/github.com/appscode/pharmer/docs/reference"
	fmt.Printf("Generating cli markdown tree in: %v\n", dir)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	doc.GenMarkdownTree(rootCmd, dir)
}

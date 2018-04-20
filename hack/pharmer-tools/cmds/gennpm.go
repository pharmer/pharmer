package cmds

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/appscode/go/runtime"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/data/files"
	"github.com/spf13/cobra"
)

func NewCmdGenNPM() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "gennpm",
		Short:             "Generate NPM index file",
		Example:           "",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			clouds := map[string]data.CloudData{}

			dataFiles, err := files.LoadDataFiles()
			if err != nil {
				log.Fatalln(err)
			}
			for _, bytes := range dataFiles {
				var cd data.CloudData
				if err := json.Unmarshal(bytes, &cd); err != nil {
					log.Fatalln(err)
				}
				clouds[cd.Name] = cd
			}

			content, err := json.MarshalIndent(clouds, "", "  ")
			if err != nil {
				log.Fatalln(err)
			}
			err = ioutil.WriteFile(runtime.GOPath()+"/src/github.com/pharmer/pharmer/hack/pharmer-data/index.json", content, 0644)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	return cmd
}

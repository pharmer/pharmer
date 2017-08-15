package cmds

import (
	"os"

	"github.com/appscode/go-term"
	otx "github.com/appscode/pharmer/config"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newCmdGet() *cobra.Command {
	setCmd := &cobra.Command{
		Use:               "get-contexts",
		Short:             "List available contexts",
		Example:           "pharmer config get-contexts",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				cmd.Help()
				os.Exit(1)
			}
			getContexts(otx.GetConfigPath(cmd))
		},
	}
	return setCmd
}

func getContexts(configPath string) {
	config, err := otx.LoadConfig(configPath)
	term.ExitOnError(err)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_CENTRE)
	table.SetHeader([]string{"CURRENT", "NAME", "PROVIDER"})
	ctx := config.CurrentContext
	for _, pharmerCtx := range config.Contexts {
		if pharmerCtx.Name == ctx {
			table.Append([]string{"*", pharmerCtx.Name, pharmerCtx.Provider})
		} else {
			table.Append([]string{"", pharmerCtx.Name, pharmerCtx.Provider})
		}
	}
	table.Render()
}

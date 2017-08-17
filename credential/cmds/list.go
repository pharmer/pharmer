package cmds

import (
	"os"

	proto "github.com/appscode/api/credential/v1beta1"
	"github.com/appscode/appctl/pkg/util/timeutil"
	term "github.com/appscode/go-term"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func NewCmdCredentialList() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list",
		Short:             "List cloud credentials",
		Example:           `appctl credential list`,
		DisableAutoGenTag: true,
		Run:               listCredential,
	}
	return cmd
}

func listCredential(cmd *cobra.Command, args []string) {
	//c := config.ClientOrDie()
	//resp, err := c.CloudCredential().List(c.Context(), &dtypes.VoidRequest{})
	//util.PrintStatus(err)

	var resp proto.CredentialListResponse
	if len(resp.Credentials) <= 0 {
		term.Infoln("Credential not found")
		os.Exit(0)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_CENTRE)
	table.SetRowLine(true)
	table.SetColWidth(40)
	table.SetHeader([]string{"Credential Name", "Provider", "Details"})
	for _, cred := range resp.Credentials {
		table.Append([]string{cred.Name, cred.Provider, cred.Information + " at " + timeutil.Format(cred.ModifiedAt)})
	}
	table.Render()
}

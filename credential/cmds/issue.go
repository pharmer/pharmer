package cmds

import (
	"os"

	proto "github.com/appscode/api/credential/v1beta1"
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
)

func NewCmdIssue() *cobra.Command {
	var req proto.CredentialCreateRequest
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create credential for cloud providers (example, AWS, Google Cloud Platform)",
		Example: `appctl credential create -p aws mycred
appctl credential create -p azure mycred
appctl credential create -p gce mycred`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			req.Name = args[0]
			if req.Name == "" {
				cmd.Help()
				os.Exit(1)
			}

			flags.EnsureRequiredFlags(cmd, "provider")
			issueCredential(&req)
		},
	}
	cmd.Flags().StringVarP(&req.Provider, "provider", "p", "", "Cloud provider name (e.g., aws, gce, azure)")

	return cmd
}

func issueCredential(req *proto.CredentialCreateRequest) {
	//if termutil.Isatty(os.Stdin.Fd()) {
	//	if req.Provider == "aws" {
	//		cloud.IssueAWSCredential(req)
	//	} else if req.Provider == "gce" {
	//		cloud.IssueGCECredential(req)
	//	} else if req.Provider == "azure" {
	//		cloud.IssueAzureCredential(req)
	//	}
	//} else {
	//	reader := bufio.NewReader(os.Stdin)
	//	credentialData, err := ioutil.ReadAll(reader)
	//	term.ExitOnError(err)
	//	req.Data, err = credential.ParseCloudCredential(string(credentialData), req.Provider)
	//	if err != nil {
	//		term.Fatalln("Failed to parse credentilal")
	//	}
	//
	//	//c := config.ClientOrDie()
	//	//_, err = c.CloudCredential().Create(c.Context(), req)
	//	//util.PrintStatus(err)
	//}
	//term.Successln("Credential created successfully!")
}

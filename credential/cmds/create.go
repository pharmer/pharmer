package cmds

import (
	"bufio"
	"io/ioutil"
	"os"

	termutil "github.com/andrew-d/go-termutil"
	proto "github.com/appscode/api/credential/v1beta1"
	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/credential"
	"github.com/spf13/cobra"
)

func NewCmdCredentialCreate() *cobra.Command {
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
			createCredential(&req)
		},
	}
	cmd.Flags().StringVarP(&req.Provider, "provider", "p", "", "Cloud provider name (e.g., aws, gce, azure)")

	return cmd
}

func createCredential(req *proto.CredentialCreateRequest) {
	if termutil.Isatty(os.Stdin.Fd()) {
		if req.Provider == "aws" {
			credential.CreateAWSCredential(req)
		} else if req.Provider == "gce" {
			credential.CreateGCECredential(req)
		} else if req.Provider == "azure" {
			credential.CreateAzureCredential(req)
		}
	} else {
		reader := bufio.NewReader(os.Stdin)
		credentialData, err := ioutil.ReadAll(reader)
		term.ExitOnError(err)
		req.Data, err = credential.ParseCloudCredential(string(credentialData), req.Provider)
		if err != nil {
			term.Fatalln("Failed to parse credentilal")
		}

		//c := config.ClientOrDie()
		//_, err = c.CloudCredential().Create(c.Context(), req)
		//util.PrintStatus(err)
	}
	term.Successln("Credential created successfully!")
}

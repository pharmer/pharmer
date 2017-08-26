package cmds

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
)

func NewCmdUpdate() *cobra.Command {
	var name string
	var provider string
	var cred string
	var file string

	cmd := &cobra.Command{
		Use:               "update",
		Short:             "Update an existing cloud credential",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "name", "provider")
			flags.EnsureAlterableFlags(cmd, "credential", "file-path")
			updateCredential(name, provider, cred, file)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Credential name")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Cloud provider name")
	cmd.Flags().StringVarP(&cred, "credential", "c", "", "Credential data")
	cmd.Flags().StringVarP(&file, "file-path", "f", "", "Credential file path")

	return cmd
}

func updateCredential(name, provider, cred, file string) {
	//var credentialData string
	//var err error
	//if file == "" {
	//	credentialData = cred
	//} else {
	//	credentialData, err = io.ReadFile(file)
	//	if err != nil {
	//		term.ExitOnError(err)
	//	}
	//}
	//parsedCredential, err := credential.ParseCloudCredential(credentialData, provider)
	//if err != nil {
	//	term.Fatalln("Failed to parse credentilal")
	//}
	//fmt.Println(parsedCredential)
	//
	////c := config.ClientOrDie()
	////_, err = c.CloudCredential().Update(c.Context(), &proto.CredentialUpdateRequest{
	////	Name:     name,
	////	Provider: provider,
	////	Data:     parsedCredential,
	////})
	////util.PrintStatus(err)
	////term.Successln("Credential updated successfully!")
}

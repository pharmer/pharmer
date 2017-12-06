package cmds

import (
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/credential"
	cc "github.com/pharmer/pharmer/credential/cloud"
	"github.com/pharmer/pharmer/data/files"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdCreateCredential() *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCredential,
		Aliases: []string{
			api.ResourceTypeCredential,
			api.ResourceCodeCredential,
			api.ResourceKindCredential,
		},
		Short:             "Create  credential object",
		Example:           `pharmer create credential`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				term.Fatalln("Missing credential name.")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple credential name provided.")
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if err := runCreateCredential(ctx, cmd, args); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringP("provider", "p", "", "Name of the Cloud provider")
	cmd.Flags().BoolP("from-env", "l", false, "Load credential data from ENV.")
	cmd.Flags().StringP("from-file", "f", "", "Load credential data from file")
	cmd.Flags().Bool("issue", false, "Issue credential")

	return cmd
}

func runCreateCredential(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Get Cloud provider
	provider, _ := cmd.Flags().GetString("provider")
	if provider == "" {
		options := files.CredentialProviders().List()
		prompt := &survey.Select{
			Message:  "Choose a Cloud provider:",
			Options:  options,
			PageSize: len(options),
		}
		survey.AskOne(prompt, &provider, nil)
	} else {
		if !files.CredentialProviders().Has(provider) {
			return errors.New("Unknown Cloud provider")
		}
	}

	issue, _ := cmd.Flags().GetBool("issue")
	if issue {
		if provider == "GoogleCloud" {
			cc.IssueGCECredential(args[0])
		} else if strings.ToLower(provider) == "azure" {
			cred, err := cc.IssueAzureCredential(args[0])
			if err != nil {
				term.Fatalln(err)
			}
			_, err = cloud.Store(ctx).Credentials().Create(cred)
			if err != nil {
				term.Fatalln(err)
			}
		} else {
			return fmt.Errorf("can't issue credential for provider %s", provider)
		}
		return nil
	}

	cred := &api.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:              args[0],
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.CredentialSpec{
			Provider: provider,
			Data:     make(map[string]string),
		},
	}

	var err error
	var commonSpec credential.CommonSpec

	if fromEnv, _ := cmd.Flags().GetBool("from-env"); fromEnv {
		commonSpec.LoadFromEnv()
	} else if fromFile, _ := cmd.Flags().GetString("from-file"); fromFile != "" {
		if commonSpec, err = credential.LoadCredentialDataFromJson(provider, fromFile); err != nil {
			return err
		}
	} else {
		cf, _ := files.GetCredentialFormat(provider)
		commonSpec.Data = make(map[string]string)
		for _, f := range cf.Fields {
			if f.Input == "password" {
				commonSpec.Data[f.JSON] = term.ReadMasked(f.Label)
			} else {
				commonSpec.Data[f.JSON] = term.Read(f.Label)
			}
		}
	}

	cred.Spec.Data = commonSpec.Data
	_, err = cloud.Store(ctx).Credentials().Create(cred)
	return err
}

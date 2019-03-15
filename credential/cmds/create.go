package cmds

import (
	"strings"
	"time"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/credential"
	cc "github.com/pharmer/pharmer/credential/cloud"
	"github.com/pharmer/pharmer/credential/cmds/options"
	"github.com/pharmer/pharmer/data/files"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	survey "gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdCreateCredential() *cobra.Command {
	opts := options.NewCredentialCreateConfig()
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
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if err := RunCreateCredential(ctx, opts); err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunCreateCredential(ctx context.Context, opts *options.CredentialCreateConfig) error {
	// Get Cloud provider
	provider := opts.Provider
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

	issue := opts.Issue
	if issue {
		if provider == "GoogleCloud" {
			cc.IssueGCECredential(opts.Name)
		} else if strings.ToLower(provider) == "azure" {
			cred, err := cc.IssueAzureCredential(opts.Name)
			if err != nil {
				term.Fatalln(err)
			}
			_, err = cloud.Store(ctx).Owner(opts.Owner).Credentials().Create(cred)
			if err != nil {
				term.Fatalln(err)
			}
		} else {
			return errors.Errorf("can't issue credential for provider %s", provider)
		}
		return nil
	}

	cred := &api.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:              opts.Name,
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.CredentialSpec{
			Provider: provider,
			Data:     make(map[string]string),
		},
	}

	var err error
	var commonSpec credential.CommonSpec
	commonSpec.Provider = provider

	if opts.FromEnv {
		commonSpec.LoadFromEnv()
	} else if opts.FromFile != "" {
		if commonSpec, err = credential.LoadCredentialDataFromJson(provider, opts.FromFile); err != nil {
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
	_, err = cloud.Store(ctx).Owner(opts.Owner).Credentials().Create(cred)
	return err
}

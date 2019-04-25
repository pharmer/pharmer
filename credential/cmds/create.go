package cmds

import (
	"strings"
	"time"

	"github.com/appscode/go/term"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	cc "github.com/pharmer/cloud/pkg/credential/cloud"
	"github.com/pharmer/cloud/pkg/providers"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/credential/cmds/options"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	survey "gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func NewCmdCreateCredential() *cobra.Command {
	opts := options.NewCredentialCreateConfig()
	cmd := &cobra.Command{
		Use: cloudapi.ResourceNameCredential,
		Aliases: []string{
			cloudapi.ResourceTypeCredential,
			cloudapi.ResourceCodeCredential,
			cloudapi.ResourceKindCredential,
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
		options := providers.List()
		prompt := &survey.Select{
			Message:  "Choose a Cloud provider:",
			Options:  options,
			PageSize: len(options),
		}
		survey.AskOne(prompt, &provider, nil)
	} else {
		if !sets.NewString(providers.List()...).Has(provider) {
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

	cred := &cloudapi.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:              opts.Name,
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: cloudapi.CredentialSpec{
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
		i, _ := providers.NewCloudProvider(providers.Options{Provider: provider})
		cf := i.ListCredentialFormats()[0]
		commonSpec.Data = make(map[string]string)
		for _, f := range cf.Spec.Fields {
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

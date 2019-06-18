package cmds

import (
	"strings"
	"time"

	"github.com/appscode/go/term"
	"github.com/pharmer/cloud/pkg/apis"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	cc "github.com/pharmer/cloud/pkg/credential/cloud"
	"github.com/pharmer/cloud/pkg/providers"
	options2 "github.com/pharmer/pharmer/cmds/credential/cmds/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func NewCmdCreateCredential() *cobra.Command {
	opts := options2.NewCredentialCreateConfig()
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

			storeProvider, err := store.GetStoreProvider(cmd, opts.Owner)
			if err != nil {
				term.Fatalln(err)
			}

			if err := runCreateCredential(storeProvider.Credentials(), opts); err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runCreateCredential(credentialStore store.CredentialStore, opts *options2.CredentialCreateConfig) error {
	// Get Cloud provider
	provider := opts.Provider
	if provider == "" {
		opts := providers.List()
		prompt := &survey.Select{
			Message:  "Choose a Cloud provider:",
			Options:  opts,
			PageSize: len(opts),
		}
		err := survey.AskOne(prompt, &provider, nil)
		if err != nil {
			return err
		}
	} else if !sets.NewString(providers.List()...).Has(provider) {
		return errors.New("Unknown Cloud provider")
	}

	issue := opts.Issue
	if issue {
		if provider == apis.GCE {
			err := cc.IssueGCECredential(opts.Name)
			if err != nil {
				return err
			}
		} else if strings.ToLower(provider) == apis.Azure {
			cred, err := cc.IssueAzureCredential(opts.Name)
			if err != nil {
				term.Fatalln(err)
			}
			_, err = credentialStore.Create(cred)
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
		commonSpec.LoadFromEnv(credential.GetFormat(provider))
	} else if opts.FromFile != "" {
		if commonSpec, err = credential.LoadCredentialDataFromJson(provider, opts.FromFile); err != nil {
			return err
		}
	} else {
		cf := credential.GetFormat(provider)
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
	_, err = credentialStore.Create(cred)
	return err
}

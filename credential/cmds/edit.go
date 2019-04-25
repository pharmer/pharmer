package cmds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/appscode/go/term"
	"github.com/ghodss/yaml"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/credential/cmds/options"
	"github.com/pharmer/pharmer/utils"
	"github.com/pharmer/pharmer/utils/editor"
	"github.com/pharmer/pharmer/utils/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func NewCmdEditCredential(out, outErr io.Writer) *cobra.Command {
	opts := options.NewCredentialEditConfig()
	cmd := &cobra.Command{
		Use: cloudapi.ResourceNameCredential,
		Aliases: []string{
			cloudapi.ResourceTypeCredential,
			cloudapi.ResourceKindCredential,
		},
		Short:             "Edit a cloud Credential",
		Example:           `pharmer edit credential`,
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

			if err := RunUpdateCredential(ctx, opts, out, outErr); err != nil {
				term.Fatalln(err)
			}
		},
	}

	opts.AddFlags(cmd.Flags())
	return cmd
}

func RunUpdateCredential(ctx context.Context, opts *options.CredentialEditConfig, out, errOut io.Writer) error {

	// If file is provided
	if opts.File != "" {
		fileName := opts.File

		var local *cloudapi.Credential
		if err := cloud.ReadFileAs(fileName, &local); err != nil {
			return err
		}

		updated, err := cloud.Store(ctx).Owner(opts.Owner).Credentials().Get(local.Name)
		if err != nil {
			return err
		}
		updated.ObjectMeta = local.ObjectMeta
		updated.Spec = local.Spec

		original, err := cloud.Store(ctx).Owner(opts.Owner).Credentials().Get(updated.Name)
		if err != nil {
			return err
		}
		if err := updateCredential(ctx, original, updated, opts.Owner); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`credential "%s" replaced`, original.Name))
		return nil
	}

	credential := opts.Name

	original, err := cloud.Store(ctx).Owner(opts.Owner).Credentials().Get(credential)
	if err != nil {
		return err
	}

	// Check if flags are provided to update
	if opts.DoNotDelete {
		updated, err := cloud.Store(ctx).Owner(opts.Owner).Credentials().Get(credential)
		if err != nil {
			return err
		}

		// Set flag values in updated object

		if err := updateCredential(ctx, original, updated, opts.Owner); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`credential "%s" updated`, original.Name))
		return nil
	}

	return editCredential(ctx, opts, original, errOut)
}

func editCredential(ctx context.Context, opts *options.CredentialEditConfig, original *cloudapi.Credential, errOut io.Writer) error {

	o, err := printer.NewEditPrinter(opts.Output)
	if err != nil {
		return err
	}

	edit := editor.NewDefaultEditor()

	containsError := false

	editFn := func() error {
		var (
			results      = editor.EditResults{}
			originalByte = []byte{}
			edited       = []byte{}
			file         string
		)

		for {
			objToEdit := original

			buf := &bytes.Buffer{}
			var w io.Writer = buf

			if o.AddHeader {
				results.Header.WriteTo(w)
			}

			if !containsError {
				if err := o.Printer.PrintObj(objToEdit, w); err != nil {
					return editor.PreservedFile(err, results.File, errOut)
				}
				originalByte = buf.Bytes()
			} else {
				buf.Write(editor.ManualStrip(edited))
			}

			// launch the editor
			editedDiff := edited
			edited, file, err = edit.LaunchTempFile(fmt.Sprintf("%s-edit-", filepath.Base(os.Args[0])), o.Ext, buf)
			if err != nil {
				return editor.PreservedFile(err, results.File, errOut)
			}

			if containsError {
				if bytes.Equal(editor.StripComments(editedDiff), editor.StripComments(edited)) {
					return editor.PreservedFile(errors.Errorf("%s", "Edit cancelled, no valid changes were saved."), file, errOut)
				}
			}

			// cleanup any file from the previous pass
			if len(results.File) > 0 {
				os.Remove(results.File)
			}

			// Compare content without comments
			if bytes.Equal(editor.StripComments(originalByte), editor.StripComments(edited)) {
				fmt.Fprintln(errOut, "Edit cancelled, no changes made.")
				return nil
			}

			var updated *cloudapi.Credential
			err = yaml.Unmarshal(editor.StripComments(edited), &updated)
			if err != nil {
				containsError = true
				results.Header.Reasons = append(results.Header.Reasons, editor.EditReason{Head: fmt.Sprintf("The edited file had a syntax error: %v", err)})
				continue
			}

			containsError = false

			if err := updateCredential(ctx, original, updated, opts.Owner); err != nil {
				return err
			}

			os.Remove(file)
			term.Println(fmt.Sprintf(`credential "%s" edited`, original.Name))
			return nil
		}
	}

	return editFn()
}

func updateCredential(ctx context.Context, original, updated *cloudapi.Credential, owner string) error {
	originalByte, err := yaml.Marshal(original)
	if err != nil {
		return err
	}
	originalJS, err := kyaml.ToJSON(originalByte)
	if err != nil {
		return err
	}

	updatedByte, err := yaml.Marshal(updated)
	if err != nil {
		return err
	}
	updatedJS, err := kyaml.ToJSON(updatedByte)
	if err != nil {
		return err
	}

	// Compare content without comments
	if bytes.Equal(editor.StripComments(originalByte), editor.StripComments(updatedByte)) {
		return errors.New("No changes made.")
	}

	preconditions := utils.GetPreconditionFunc("")
	patch, err := strategicpatch.CreateTwoWayMergePatch(originalJS, updatedJS, updated, preconditions...)
	if err != nil {
		if mergepatch.IsPreconditionFailed(err) {
			return editor.PreconditionFailedError()
		}
		return err
	}

	conditionalPreconditions := utils.GetConditionalPreconditionFunc(cloudapi.ResourceKindCredential)
	err = utils.CheckConditionalPrecondition(patch, conditionalPreconditions...)
	if err != nil {
		if utils.IsPreconditionFailed(err) {
			return editor.ConditionalPreconditionFailedError(cloudapi.ResourceKindCredential)
		}
		return err
	}

	_, err = cloud.Store(ctx).Owner(owner).Credentials().Update(updated)
	if err != nil {
		return err
	}
	return nil
}

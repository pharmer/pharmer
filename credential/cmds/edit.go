package cmds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/appscode/go-term"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/utils/editor"
	"github.com/appscode/pharmer/utils/printer"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"github.com/appscode/pharmer/utils"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func NewCmdEditCredential(out, outErr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCredential,
		Aliases: []string{
			api.ResourceTypeCredential,
			api.ResourceKindCredential,
		},
		Short:             "Edit a cloud Credentials",
		Example:           `pharmer edit credential <credential_name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				term.Fatalln("Missing credential name")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple credential name provided.")
			}
			credentialName := args[0]

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if err := RunEditCredential(ctx, cmd, out, outErr, credentialName); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringP("output", "o", "yaml", "Output format. One of: yaml|json.")
	return cmd
}

func RunEditCredential(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, credentialName string) error {

	o, err := printer.NewEditPrinter(cmd)
	if err != nil {
		return err
	}

	credential, err := cloud.Store(ctx).Credentials().Get(credentialName)
	if err != nil {
		return err
	}

	edit := editor.NewDefaultEditor()

	containsError := false

	editFn := func() error {
		var (
			results  = editor.EditResults{}
			original = []byte{}
			edited   = []byte{}
			file     string
		)

		for {

			originalObj := credential
			objToEdit := originalObj

			buf := &bytes.Buffer{}
			var w io.Writer = buf

			if o.AddHeader {
				results.Header.WriteTo(w)
			}

			if !containsError {
				if err := o.Printer.PrintObj(objToEdit, w); err != nil {
					return editor.PreservedFile(err, results.File, errOut)
				}
				original = buf.Bytes()
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
					return editor.PreservedFile(fmt.Errorf("%s", "Edit cancelled, no valid changes were saved."), file, errOut)
				}
			}

			// cleanup any file from the previous pass
			if len(results.File) > 0 {
				os.Remove(results.File)
			}

			// Compare content without comments
			if bytes.Equal(editor.StripComments(original), editor.StripComments(edited)) {
				fmt.Fprintln(errOut, "Edit cancelled, no changes made.")
				return nil
			}

			var updatedCredential *api.Credential
			err = yaml.Unmarshal(editor.StripComments(edited), &updatedCredential)
			if err != nil {
				containsError = true
				results.Header.Reasons = append(results.Header.Reasons, editor.EditReason{Head: fmt.Sprintf("The edited file had a syntax error: %v", err)})
				continue
			}

			containsError = false

			originalByte, err := yaml.Marshal(credential)
			if err != nil {
				return editor.PreservedFile(err, results.File, errOut)
			}
			originalJS, err := kyaml.ToJSON(originalByte)
			if err != nil {
				return err
			}

			editedJS := editor.StripComments(edited)

			preconditions := utils.GetPreconditionFunc("")
			patch, err := strategicpatch.CreateTwoWayMergePatch(originalJS, editedJS, updatedCredential, preconditions...)
			if err != nil {
				if mergepatch.IsPreconditionFailed(err) {
					return editor.PreconditionFailedError()
				}
				return err
			}

			conditionalPreconditions := utils.GetConditionalPreconditionFunc(api.ResourceKindCredential)
			err = utils.CheckConditionalPrecondition(patch, conditionalPreconditions...)
			if err != nil {
				if utils.IsPreconditionFailed(err) {
					return editor.ConditionalPreconditionFailedError(api.ResourceKindCredential)
				}
				return err
			}

			_, err = cloud.Store(ctx).Credentials().Update(updatedCredential)
			if err != nil {
				return editor.PreservedFile(err, results.File, errOut)
			}

			os.Remove(file)
			term.Printf(`credential "%s" edited\n`, credentialName)
			return nil
		}
	}

	return editFn()
}

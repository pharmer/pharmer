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

func NewCmdEditNodeGroup(out, outErr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Edit a Kubernetes cluster NodeGroup",
		Example:           `pharmer edit nodegroup -k <cluster_name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				term.Fatalln("Missing nodegroup name")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple nodegroup name provided.")
			}
			nodeGroupName := args[0]

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if err := RunEditNodeGroup(ctx, cmd, out, outErr, nodeGroupName); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	cmd.Flags().StringP("output", "o", "yaml", "Output format. One of: yaml|json.")
	return cmd
}

func RunEditNodeGroup(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, nodeGroupName string) error {

	o, err := printer.NewEditPrinter(cmd)
	if err != nil {
		return err
	}

	clusterName, err := cmd.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	nodeGroup, err := cloud.Store(ctx).NodeGroups(clusterName).Get(nodeGroupName)
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

			originalObj := nodeGroup
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

			var updatedNodeGroup *api.NodeGroup
			err = yaml.Unmarshal(editor.StripComments(edited), &updatedNodeGroup)
			if err != nil {
				containsError = true
				results.Header.Reasons = append(results.Header.Reasons, editor.EditReason{Head: fmt.Sprintf("The edited file had a syntax error: %v", err)})
				continue
			}

			containsError = false

			originalByte, err := yaml.Marshal(nodeGroup)
			if err != nil {
				return editor.PreservedFile(err, results.File, errOut)
			}
			originalJS, err := kyaml.ToJSON(originalByte)
			if err != nil {
				return err
			}

			editedJS := editor.StripComments(edited)

			preconditions := utils.GetPreconditionFunc("")
			patch, err := strategicpatch.CreateTwoWayMergePatch(originalJS, editedJS, updatedNodeGroup, preconditions...)
			if err != nil {
				if mergepatch.IsPreconditionFailed(err) {
					return editor.PreconditionFailedError()
				}
				return err
			}

			conditionalPreconditions := utils.GetConditionalPreconditionFunc(api.ResourceKindNodeGroup)
			err = utils.CheckConditionalPrecondition(patch, conditionalPreconditions...)
			if err != nil {
				if utils.IsPreconditionFailed(err) {
					return editor.ConditionalPreconditionFailedError(api.ResourceKindNodeGroup)
				}
				return err
			}

			_, err = cloud.Store(ctx).NodeGroups(clusterName).Update(updatedNodeGroup)
			if err != nil {
				return editor.PreservedFile(err, results.File, errOut)
			}

			os.Remove(file)
			term.Printf(`nodegroup "%s" edited\n`, nodeGroupName)
			return nil
		}
	}

	return editFn()
}

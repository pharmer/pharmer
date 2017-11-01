package cmds

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/appscode/go-term"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/utils"
	"github.com/appscode/pharmer/utils/editor"
	"github.com/appscode/pharmer/utils/printer"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func NewCmdEditCluster(out, outErr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Edit cluster object",
		Example:           `pharmer edit cluster`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if err := runUpdateCluster(ctx, cmd, out, outErr, args); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringP("file", "f", "", "Load cluster data from file")
	//TODO: Add necessary flags that will be used for update
	cmd.Flags().BoolP("do-not-delete", "", false, "Set do not delete flag")
	cmd.Flags().StringP("output", "o", "yaml", "Output format. One of: yaml|json.")
	return cmd
}

func runUpdateCluster(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, args []string) error {
	// If file is provided
	if cmd.Flags().Changed("file") {
		fileName, err := cmd.Flags().GetString("file")
		if err != nil {
			return err
		}
		var updated *api.Cluster
		if err := cloud.ReadFileAs(fileName, &updated); err != nil {
			return err
		}

		original, err := cloud.Store(ctx).Clusters().Get(updated.Name)
		if err != nil {
			return err
		}
		if err := updateCluster(ctx, original, updated); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`cluster "%s" replaced`, original.Name))
		return nil
	}

	if len(args) == 0 {
		return errors.New("Missing cluster name")
	}
	if len(args) > 1 {
		return errors.New("Multiple cluster name provided.")
	}
	clusterName := args[0]

	original, err := cloud.Store(ctx).Clusters().Get(clusterName)
	if err != nil {
		return err
	}

	// Check if flags are provided to update
	// TODO: Provide list of flag names. If any of them is provided, update
	if utils.CheckAlterableFlags(cmd, "do-not-delete") {
		updated, err := cloud.Store(ctx).Clusters().Get(clusterName)
		if err != nil {
			return err
		}

		if cmd.Flags().Changed("do-not-delete") {
			doNotDelete, err := cmd.Flags().GetBool("do-not-delete")
			if err != nil {
				return err
			}
			updated.Spec.DoNotDelete = doNotDelete
		}

		//TODO: Check provided flags, and set value

		if err := updateCluster(ctx, original, updated); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`cluster "%s" updated`, original.Name))
		return nil
	}

	return editCluster(ctx, cmd, original, errOut)
}

func editCluster(ctx context.Context, cmd *cobra.Command, original *api.Cluster, errOut io.Writer) error {

	o, err := printer.NewEditPrinter(cmd)
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
					return editor.PreservedFile(fmt.Errorf("%s", "Edit cancelled, no valid changes were saved."), file, errOut)
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

			var updated *api.Cluster
			err = yaml.Unmarshal(editor.StripComments(edited), &updated)
			if err != nil {
				containsError = true
				results.Header.Reasons = append(results.Header.Reasons, editor.EditReason{Head: fmt.Sprintf("The edited file had a syntax error: %v", err)})
				continue
			}

			containsError = false

			if err := updateCluster(ctx, original, updated); err != nil {
				return err
			}

			os.Remove(file)
			term.Println(fmt.Sprintf(`cluster "%s" edited`, original.Name))
			return nil
		}
	}

	return editFn()
}

func updateCluster(ctx context.Context, original, updated *api.Cluster) error {
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

	conditionalPreconditions := utils.GetConditionalPreconditionFunc(api.ResourceKindCluster)
	err = utils.CheckConditionalPrecondition(patch, conditionalPreconditions...)
	if err != nil {
		if utils.IsPreconditionFailed(err) {
			return editor.ConditionalPreconditionFailedError(api.ResourceKindCluster)
		}
		return err
	}

	_, err = cloud.Store(ctx).Clusters().Update(updated)
	if err != nil {
		return err
	}
	return nil
}

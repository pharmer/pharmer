package cloud

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/appscode/go/term"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/utils"
	"pharmer.dev/pharmer/utils/editor"
	"pharmer.dev/pharmer/utils/printer"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func NewCmdEditNodeGroup(out, outErr io.Writer) *cobra.Command {
	opts := options.NewNodeGroupEditConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Edit a Kubernetes cluster NodeGroup",
		Example:           `pharmer edit nodegroup`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			if err := RunUpdateNodeGroup(storeProvider.MachineSet(opts.ClusterName), opts, outErr); err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunUpdateNodeGroup(machinesetStore store.MachineSetStore, opts *options.NodeGroupEditConfig, errOut io.Writer) error {
	// If file is provided
	if opts.File != "" {
		fileName := opts.File

		var local *clusterv1.MachineSet
		if err := cloud.ReadFileAs(fileName, &local); err != nil {
			return err
		}

		updated, err := machinesetStore.Get(local.Name)
		if err != nil {
			return err
		}
		updated.ObjectMeta = local.ObjectMeta
		updated.Spec = local.Spec

		original, err := machinesetStore.Get(updated.Name)
		if err != nil {
			return err
		}
		if err := UpdateNodeGroup(machinesetStore, original, updated); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`nodegroup "%s" replaced`, original.Name))
		return nil
	}

	original, err := machinesetStore.Get(opts.NgName)
	if err != nil {
		return err
	}

	// Check if flags are provided to update
	if opts.DoNotDelete {
		updated, err := machinesetStore.Get(opts.NgName)
		if err != nil {
			return err
		}

		if err := UpdateNodeGroup(machinesetStore, original, updated); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`nodegroup "%s" updated`, original.Name))
		return nil
	}

	return editNodeGroup(machinesetStore, opts, original, errOut)
}

func editNodeGroup(machinesetStore store.MachineSetStore, opts *options.NodeGroupEditConfig, original *clusterv1.MachineSet, errOut io.Writer) error {

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
				_, err = results.Header.WriteTo(w)
				if err != nil {
					return err
				}
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

			var updated *clusterv1.MachineSet
			err = yaml.Unmarshal(editor.StripComments(edited), &updated)
			if err != nil {
				containsError = true
				results.Header.Reasons = append(results.Header.Reasons, editor.EditReason{Head: fmt.Sprintf("The edited file had a syntax error: %v", err)})
				continue
			}

			containsError = false

			if err := UpdateNodeGroup(machinesetStore, original, updated); err != nil {
				return err
			}

			os.Remove(file)
			term.Println(fmt.Sprintf(`nodegroup "%s" edited`, original.Name))
			return nil
		}
	}

	return editFn()
}

func UpdateNodeGroup(machinesetStore store.MachineSetStore, original, updated *clusterv1.MachineSet) error {
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
		return errors.New("no changes made")
	}

	preconditions := utils.GetPreconditionFunc()
	patch, err := strategicpatch.CreateTwoWayMergePatch(originalJS, updatedJS, updated, preconditions...)
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

	_, err = machinesetStore.Update(updated)
	if err != nil {
		return err
	}
	return nil
}

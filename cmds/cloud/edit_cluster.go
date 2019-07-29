package cloud

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/appscode/go/term"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/utils"
	"pharmer.dev/pharmer/utils/editor"
	"pharmer.dev/pharmer/utils/printer"
)

func NewCmdEditCluster(out, outErr io.Writer) *cobra.Command {
	opts := options.NewClusterEditConfig()
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
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			if err := runUpdateCluster(storeProvider.Clusters(), opts, outErr); err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runUpdateCluster(clusterStore store.ClusterStore, opts *options.ClusterEditConfig, errOut io.Writer) error {
	// If file is provided
	if opts.File != "" {
		fileName := opts.File

		var local *api.Cluster
		if err := cloud.ReadFileAs(fileName, &local); err != nil {
			return err
		}

		updated, err := clusterStore.Get(local.Name)
		if err != nil {
			return err
		}
		updated.ObjectMeta = local.ObjectMeta
		updated.Spec = local.Spec

		original, err := clusterStore.Get(updated.Name)
		if err != nil {
			return err
		}
		if err := UpdateCluster(clusterStore, original, updated); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`cluster "%s" replaced`, original.Name))
		return nil
	}

	original, err := clusterStore.Get(opts.ClusterName)
	if err != nil {
		return err
	}

	// Check if flags are provided to update
	// TODO: Provide list of flag names. If any of them is provided, update
	if opts.CheckForUpdateFlags() {
		updated, err := clusterStore.Get(opts.ClusterName)
		if err != nil {
			return err
		}

		//TODO: Check provided flags, and set value
		if opts.KubernetesVersion != "" {
			updated.Spec.Config.KubernetesVersion = opts.KubernetesVersion
		}

		if err := UpdateCluster(clusterStore, original, updated); err != nil {
			return err
		}
		term.Println(fmt.Sprintf(`cluster "%s" updated`, original.Name))
		return nil
	}

	return editCluster(clusterStore, opts, original, errOut)
}

func editCluster(clusterStore store.ClusterStore, opts *options.ClusterEditConfig, original *api.Cluster, errOut io.Writer) error {
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
				err = os.Remove(results.File)
				if err != nil {
					return err
				}
			}

			// Compare content without comments
			if bytes.Equal(editor.StripComments(originalByte), editor.StripComments(edited)) {
				_, err = fmt.Fprintln(errOut, "Edit cancelled, no changes made.")
				return err
			}

			var updated *api.Cluster
			err = yaml.Unmarshal(editor.StripComments(edited), &updated)
			if err != nil {
				containsError = true
				results.Header.Reasons = append(results.Header.Reasons, editor.EditReason{Head: fmt.Sprintf("The edited file had a syntax error: %v", err)})
				continue
			}

			containsError = false

			if err := UpdateCluster(clusterStore, original, updated); err != nil {
				return err
			}

			term.Println(fmt.Sprintf(`cluster "%s" edited`, original.Name))
			return os.Remove(file)
		}
	}

	return editFn()
}

func UpdateCluster(clusterStore store.ClusterStore, original, updated *api.Cluster) error {
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

	conditionalPreconditions := utils.GetConditionalPreconditionFunc(api.ResourceKindCluster)
	err = utils.CheckConditionalPrecondition(patch, conditionalPreconditions...)
	if err != nil {
		if utils.IsPreconditionFailed(err) {
			return editor.ConditionalPreconditionFailedError(api.ResourceKindCluster)
		}
		return err
	}

	_, err = updateGeneration(clusterStore, updated)
	if err != nil {
		return err
	}
	return nil
}

func updateGeneration(clusterStore store.ClusterStore, cluster *api.Cluster) (*api.Cluster, error) {
	if cluster == nil {
		return nil, errors.New("missing Cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing Cluster name")
	} else if cluster.ClusterConfig().KubernetesVersion == "" {
		return nil, errors.New("missing Cluster version")
	}

	existing, err := clusterStore.Get(cluster.Name)
	if err != nil {
		return nil, errors.Errorf("Cluster `%s` does not exist. Reason: %v", cluster.Name, err)
	}
	cluster.Status = existing.Status
	cluster.Generation = time.Now().UnixNano()

	return clusterStore.Update(cluster)
}

package cmds

import (
	"io"

	"github.com/spf13/cobra"
	"fmt"
	"strings"
	"github.com/appscode/pharmer/cloud/util"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/cloud"
	"context"
	"errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/printer"
)

func NewCmdGet(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)

			RunGet(ctx, cmd, out, errOut, args)

		},
	}

	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml|wide.")
	return cmd
}

const (
	valid_resources = `Valid resource types include:

    * all
    * cluster
    `
)

func RunGet(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(errOut, "You must specify the type of resource to get. ", valid_resources)

		return fmt.Errorf("Required resource not specified.\nSee '%s -h' for help and examples.", cmd.CommandPath())
	}

	var printAll bool = false
	resources := strings.Split(args[0], ",")
	var multipleResource = len(resources) > 1
	var slashUsed = false
	for i, r := range resources {
		if r == "all" {
			printAll = true
		} else {
			items := strings.Split(r, "/")
			if len(items) > 1 {
				slashUsed = true
			}
			kind, err := util.GetSupportedResource(items[0])
			if err != nil {
				return err
			}
			items[0] = kind
			resources[i] = strings.Join(items, "/")
		}
	}

	if multipleResource && slashUsed {
		return errors.New("arguments in resource/name form must have a single resource and name")
	}

	if slashUsed && len(args) > 1 {
		return errors.New("there is no need to specify a resource type as a separate argument when passing arguments in resource/name form")
	}

	if printAll {
		resources = util.GetAllSupportedResources()
	}

	rPrinter, err := printer.NewPrinter(cmd)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	objects := make([]string, 0)

	for _, r := range resources {
		items := strings.Split(r, "/")
		kind := items[0]
		switch kind {
		case "cluster":
			if len(items) > 1 {
				objects = []string{items[1]}
			} else {
				objects = append(objects, args[1:]...)
			}
			clusters, err := getClusterList(ctx, objects)
			if err != nil {
				return err
			}
			for _, cluster := range clusters {
				if err := rPrinter.PrintObj(cluster, w); err != nil {
					return err
				}
				if rPrinter.IsGeneric() {
					printer.PrintNewline(w)
				}
			}
		}
	}

	w.Flush()
	return nil
}

func getClusterList(ctx context.Context, args []string) (clusterList []*api.Cluster, err error) {
	if len(args) == 1 {
		cluster, er2 := cloud.Store(ctx).Clusters().Get(args[0])
		if er2 != nil {
			return nil, er2
		}
		clusterList = append(clusterList, cluster)
	} else {
		clusterList, err = cloud.Store(ctx).Clusters().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}

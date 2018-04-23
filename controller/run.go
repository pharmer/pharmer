package controller

import (
	"github.com/spf13/cobra"
	//"github.com/golang/glog"
	"github.com/kubernetes-incubator/apiserver-builder/pkg/controller"
	/*"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/util/logs"*/

	//"sigs.k8s.io/cluster-api/cloud/google"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	"sigs.k8s.io/cluster-api/pkg/controller/config"
	"sigs.k8s.io/cluster-api/pkg/controller/machine"
	"sigs.k8s.io/cluster-api/pkg/controller/sharedinformers"
	"github.com/appscode/go/term"
	"fmt"
)


func NewCmdRun() *cobra.Command  {
	s := config.ControllerConfig
	cmd := &cobra.Command{
		Use:               "run",
		Short:             "Bootstrap as a Kubernetes master or node",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			conf, err := controller.GetConfig(s.Kubeconfig)
			if err != nil {
				term.Fatalln(err)
			}
			client, err := clientset.NewForConfig(conf)
			if err != nil {
				term.Fatalln(err)
			}
			fmt.Println(client)
			//actuator, err :=
			shutdown := make(chan struct{})
			si := sharedinformers.NewSharedInformers(conf, shutdown)
			c := machine.NewMachineController(conf, si, nil)
			c.Run(shutdown)
			select {}
		},
	}
	s.AddFlags(cmd.Flags())
	return cmd
}

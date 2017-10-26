package inspector

import (
	"fmt"
	/*"io/ioutil"
	"os"
	"time"*/

	//proto "github.com/appscode/api/kubernetes/v1beta1"
	//appscodeSSH "github.com/appscode/api/ssh/v1beta1"
	//"github.com/appscode/client/cli"
	"github.com/appscode/go/errors"
	//"github.com/cenkalti/backoff"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/clientcmd"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	clientcmd_api "k8s.io/client-go/tools/clientcmd/api"
	clientcmd_v1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/kubernetes/scheme"
	. "github.com/appscode/pharmer/cloud"
	"context"
	"github.com/appscode/go-term"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Inspector struct {
	ctx     context.Context
	client        kubernetes.Interface
	cluster *api.Cluster
//	cm Interface
	config *rest.Config
	//ssh     SSHExecutor
}

func New(ctx context.Context, cluster *api.Cluster) (*Inspector, error) {
	if cluster.Spec.Cloud.CloudProvider == "" {
		return nil, fmt.Errorf("cluster %v has no provider", cluster.Name)
	}
	var err error
	if ctx, err = LoadCACertificates(ctx, cluster); err != nil {
		return nil, err
	}
	if ctx, err = LoadSSHKey(ctx, cluster); err != nil {
		return nil, err
	}
	kc, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return nil, err
	}

	adminConfig, err := GetAdminConfig(ctx, cluster)
	if err != nil {
		return nil, err
	}
	err = clientcmd_v1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	out := &clientcmd_api.Config{}
	err = scheme.Scheme.Convert(adminConfig, out, nil)
	if err != nil {
		return nil, err
	}
	clientConfig := clientcmd.NewDefaultClientConfig(*out, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	/*cm, err := GetCloudManager(cluster.Spec.Cloud.CloudProvider, ctx)
	if err != nil {
		return nil, err
	}*/

	return  &Inspector{ctx: ctx, client: kc,cluster: cluster, config: restConfig}, nil
}


func (i *Inspector) NativeCheck() error {
	if err := i.CheckHelthStatus(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := i.checkRBAC(); err != nil {
		return errors.FromErr(err).Err()
	}
	/*if err := c.checkLoadBalancer(); err != nil {
		return errors.FromErr(err).Err()
	}*/
	return nil
}


func (i *Inspector) NetworkCheck() error {
	if err := i.CheckDNSPod(); err != nil {
		return err
	}

	nodes, err := i.getNodes()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	var nodeonly = make([]string, 0)
	var masterNode apiv1.Node
	for _, n := range nodes.Items {
		if _, ok := n.ObjectMeta.Labels["node-role.kubernetes.io/master"]; !ok {
			nodeonly = append(nodeonly, n.Name)
		} else {
			masterNode = n
		}
	}
	if len(nodeonly) <= 1 {
		term.Fatalln("Need at least 2 nodes to check")
		return nil
	}
	fmt.Println(masterNode)

	defer func() {
		i.DeleteNginx()
		//i.DeleteNginxPod(pod2)
		//i.DeleteNginxService(pod1)
		//i.DeleteNginxService(pod2)
	}()

	var pods []apiv1.Pod
	if pods, err = i.InstallNginx(); err != nil {
		return err
	}

	term.Infoln("Checking Pod networks...")
	if err := i.runNodeExecutor(pods[0].Name, pods[1].Status.PodIP, defaultNamespace, pods[0].Spec.Containers[0].Name); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := i.runNodeExecutor(pods[1].Name, pods[0].Status.PodIP, defaultNamespace, pods[1].Spec.Containers[0].Name); err != nil {
		return errors.FromErr(err).Err()
	}

	//fmt.Println(i.InstallNginxService())


	/*podname1, err := i.InstallNginxPod(pod1, nodeonly[0])
	if err != nil {
		return errors.FromErr(err).Err()
	}
	podname2, err := i.InstallNginxPod(pod2, nodeonly[1])
	if err != nil {
		return errors.FromErr(err).Err()
	}

	svcIp1, err := i.InstallNginxService(pod1)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	svcIp2, err := i.InstallNginxService(pod2)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	term.Infoln("Checking networks usinng service ip...", svcIp1)
	if err := i.runNodeExecutor(podname1[0].Name, svcIp2, defaultNamespace, pod1); err != nil {
		return errors.FromErr(err).Err()
	}
	term.Infoln("Checking networks using service name...")
	if err := i.runNodeExecutor(podname1[0].Name, pod2+"."+defaultNamespace, defaultNamespace, pod1); err != nil {
		return errors.FromErr(err).Err()
	}
	term.Infoln("Checking from master")
	if err := i.runMasterExecutor(masterNode, podname1[0].Status.PodIP); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := i.runMasterExecutor(masterNode, podname2[0].Status.PodIP); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := i.runMasterExecutor(masterNode, svcIp1); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := i.runMasterExecutor(masterNode, svcIp2); err != nil {
		return errors.FromErr(err).Err()
	}*/

	return nil
}


/*
func (c *Cluster) callClusterConfigApi(req proto.ClusterClientConfigRequest) (*proto.ClusterClientConfigResponse, error) {
	var resp *proto.ClusterClientConfigResponse
	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err = client.Kubernetes().V1beta1().Cluster().ClientConfig(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	return resp, err
}

func (c *Cluster) callClusterSSHApi(req appscodeSSH.SSHGetRequest) (*appscodeSSH.SSHGetResponse, error) {
	var resp *appscodeSSH.SSHGetResponse
	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err = client.SSH().Get(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	return resp, err
}
*/

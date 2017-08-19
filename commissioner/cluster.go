package commissioner

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/client/cli"
	"github.com/appscode/errors"
	term "github.com/appscode/go-term"
	"github.com/appscode/log"
	vcs "github.com/appscode/voyager/client/clientset"
	"github.com/cenkalti/backoff"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const initialNode = 2

func (c *Cluster) ClusterCreate(version string) error {
	req := c.ClusterCreateRequestBuild()
	req.CredentialUid = c.CredentialPHID
	req.KubernetesVersion = version
	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		_, err = client.Kubernetes().V1beta1().Cluster().Create(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	if err != nil {
		return errors.FromErr(err).Err()
	}
	log.Infoln("waiting for 3 minutes to create cluster")
	waitTime(180)
	if err := c.InstallKubeConfig(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.LoadKubeClient(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.checkNodeReady(initialNode + 1); err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *Cluster) checkNodeReady(nodeNumber int) error {
	retry := 30
	for retry > 0 {
		count := 0
		nodes := &apiv1.NodeList{}
		if err := c.Kube.Client.CoreV1().RESTClient().Get().Resource("nodes").Do().Into(nodes); err != nil {
			fmt.Println(err)
		}
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					log.Infoln(node.Name, "is ready")
					count++
				}
			}
		}
		if count == nodeNumber {
			break
		}
		time.Sleep(1 * time.Minute)
		fmt.Println("Waiting for ready node")
		retry--
	}
	if retry == 0 {
		return errors.New("Nodes are not ready. Problem with creating cluster").Err()
	}
	term.Successln("All nodes are ready...")
	return nil
}

func (c *Cluster) LoadKubeClient() error {
	kc, err := c.getKubeConfig()
	if err != nil {
		fmt.Println(err)
		log.Fatalln("Failed to load Kube Config")
		return errors.FromErr(err).Err()
	}

	c.Kube = &Kube{
		Client:        clientset.NewForConfigOrDie(kc),
		Config:        kc,
		VoyagerClient: vcs.NewForConfigOrDie(kc),
	}

	return nil
}

func (c *Cluster) NativeCheck() error {
	if err := c.LoadKubeClient(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.CheckHelthStatus(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.checkRBAC(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.checkLoadBalancer(); err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *Cluster) ClusterScale() error {
	req := proto.ClusterReconfigureRequest{
		Name: c.Name,
	}
	req.Sku = getScaleSku(c.Provider)
	req.Count = 1
	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		_, err = client.Kubernetes().V1beta1().Cluster().Reconfigure(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	if err != nil {
		return errors.FromErr(err).Err()
	}
	time.Sleep(1 * time.Minute)

	if err := c.LoadKubeClient(); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.checkNodeReady(initialNode + 2); err != nil {
		return errors.FromErr(err).Err()
	}
	term.Successln("Cluster scalling up successful")

	req.Count = 0
	err = backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		_, err = client.Kubernetes().V1beta1().Cluster().Reconfigure(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.checkNodeReady(initialNode + 1); err != nil {
		return errors.FromErr(err).Err()
	}
	term.Successln("Cluster scalling down successful")

	/*resp, err = c.Kubernetes().V1beta1().Cluster().Reconfigure(c.Context(), &req)
	fmt.Println(resp, err)
	time.Sleep(15 * time.Minute)
	req.Version = "1.5.2"
	req.ApplyToMaster = true
	resp, err = c.Kubernetes().V1beta1().Cluster().Reconfigure(c.Context(), &req)
	fmt.Println(resp, err)*/
	return nil

}

func (c *Cluster) ClusterUpgrade(version string) error {
	req := proto.ClusterReconfigureRequest{
		Name:              c.Name,
		KubernetesVersion: version,
		ApplyToMaster:     true,
	}
	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		_, err = client.Kubernetes().V1beta1().Cluster().Reconfigure(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	if err != nil {
		return errors.FromErr(err).Err()
	}
	term.Successln("Cluster is upgrading to version", version)
	return nil

}

func (c *Cluster) ClusterDelete() error {
	req := proto.ClusterDeleteRequest{
		Name:  c.Name,
		Force: true,
	}

	err := backoff.Retry(func() error {
		client, err := cli.Client("")
		if err != nil {
			return err
		}
		defer client.Close()
		_, err = client.Kubernetes().V1beta1().Cluster().Delete(client.Context(), &req)
		return err
	}, NewExponentialBackOff())
	if err != nil {
		return errors.FromErr(err).Err()
	}
	term.Successln("Cluster is deleting...")
	return nil
}

func (c *Cluster) ClusterCreateRequestBuild() proto.ClusterCreateRequest {
	req := proto.ClusterCreateRequest{
		Name:        c.Name,
		Provider:    c.Provider,
		DoNotDelete: false,
	}
	var sku string = ""
	sku, req.Zone = getProviderSku(c.Provider)
	req.NodeGroups = make([]*proto.InstanceGroup, 1)
	req.NodeGroups[0] = &proto.InstanceGroup{
		Sku:   sku,
		Count: initialNode,
	}
	return req
}

func getProviderSku(provider string) (string, string) {
	switch provider {
	case "gce":
		return "n1-standard-2", "us-central1-f"
	case "aws":
		return "t2.medium", "us-west-1b"
	case "digitalocean":
		return "2gb", "nyc3"

	}
	return "", ""
}

func getScaleSku(provider string) string {
	switch provider {
	case "gce":
		return "n1-standard-1"
	case "aws":
		return "t2.large"
	case "digitalocean":
		return "4gb"
	default:
		return ""
	}
}

func waitTime(retry int) {
	for retry > 0 {
		fmt.Print(".")
		retry--
		time.Sleep(2 * time.Second)
	}
	fmt.Println()
}

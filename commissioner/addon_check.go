package commissioner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	appscodeSSH "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/errors"
	term "github.com/appscode/go-term"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/cenkalti/backoff"
	"github.com/ghodss/yaml"
	"github.com/mgutz/str"
	ini "github.com/vaughan0/go-ini"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/api"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
)

var comissionarKubeConfigPath = clientcmd.RecommendedHomeFile

const (
	defaultNamespace = "default"
	AppscodeIcinga   = "appscode-icinga"
	ENV              = ".env"
	ENVFILE          = "/srv/appscode/.env"
)

func (c *Cluster) NetworkCheck() error {
	err := c.LoadKubeClient()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	pod1 := "my-nginx-1"
	pod2 := "my-nginx-2"
	defer func() {
		c.DeleteNginxPod(pod1)
		c.DeleteNginxPod(pod2)
		c.DeleteNginxService(pod1)
		c.DeleteNginxService(pod2)
	}()

	nodes, err := c.getNodes()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	var nodeonly = make([]string, 0)
	var masterNode string = ""
	for _, n := range nodes.Items {
		if n.ObjectMeta.Labels["kubernetes.io/role"] != "master" {
			nodeonly = append(nodeonly, n.Name)
		} else {
			masterNode = n.Name
		}
	}

	podname1, err := c.InstallNginxPod(pod1, nodeonly[0])
	if err != nil {
		return errors.FromErr(err).Err()
	}
	podname2, err := c.InstallNginxPod(pod2, nodeonly[1])
	if err != nil {
		return errors.FromErr(err).Err()
	}

	log.Infoln("Checking Pod networks...")
	if err := c.runNodeExecutor(podname1[0].Name, podname2[0].Status.PodIP, defaultNamespace, pod1); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.runNodeExecutor(podname2[0].Name, podname1[0].Status.PodIP, defaultNamespace, pod2); err != nil {
		return errors.FromErr(err).Err()
	}

	svcIp1, err := c.InstallNginxService(pod1)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	svcIp2, err := c.InstallNginxService(pod2)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	fmt.Println("Checking networks usinng service ip...", svcIp1)
	if err := c.runNodeExecutor(podname1[0].Name, svcIp2, defaultNamespace, pod1); err != nil {
		return errors.FromErr(err).Err()
	}
	fmt.Println("Checking networks using service name...")
	if err := c.runNodeExecutor(podname1[0].Name, pod2+"."+defaultNamespace, defaultNamespace, pod1); err != nil {
		return errors.FromErr(err).Err()
	}
	fmt.Println("Checking from master")
	if err := c.runMasterExecutor(masterNode, podname1[0].Status.PodIP); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.runMasterExecutor(masterNode, podname2[0].Status.PodIP); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.runMasterExecutor(masterNode, svcIp1); err != nil {
		return errors.FromErr(err).Err()
	}
	if err := c.runMasterExecutor(masterNode, svcIp2); err != nil {
		return errors.FromErr(err).Err()
	}

	return nil
}

func (c *Cluster) AddonSetup() error {
	c.LoadKubeClient()
	nodes, err := c.getNodes()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	for _, node := range nodes.Items {
		if node.Name != c.Name+"-master" {
			writeENVVar("NODE_NAME", node.Name)
			//fmt.Println(node.Name)
			break
		}
	}
	c.setupIcinga()

	return nil
}

func (c *Cluster) getNodes() (*apiv1.NodeList, error) {
	nodes := &apiv1.NodeList{}
	if err := c.Kube.Client.CoreV1().RESTClient().Get().Resource("nodes").Do().Into(nodes); err != nil {
		return nodes, errors.FromErr(err).Err()
	}
	return nodes, nil
}

func (c *Cluster) setupIcinga() error {
	svc, err := c.Kube.Client.CoreV1().Services(api.NamespaceSystem).Get(AppscodeIcinga, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if svc.Spec.Type != apiv1.ServiceTypeLoadBalancer {
		svc.Spec.Type = apiv1.ServiceTypeLoadBalancer
		svc, err = c.Kube.Client.CoreV1().Services(api.NamespaceSystem).Update(svc)
		time.Sleep(1 * time.Minute)
	}
	var ip string = ""
	for ip == "" {
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			ip = svc.Status.LoadBalancer.Ingress[0].IP
		} else {
			svc, _ = c.Kube.Client.CoreV1().Services(api.NamespaceSystem).Get(AppscodeIcinga, metav1.GetOptions{})
		}
	}
	fmt.Println(ip)
	writeENVVar("ICINGA_ADDRESS", ip)

	sec, err := c.Kube.Client.CoreV1().Secrets(api.NamespaceSystem).Get(AppscodeIcinga, metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if data, found := sec.Data[ENV]; found {
		dataReader := strings.NewReader(string(data))
		secretData, err := ini.Load(dataReader)
		if err != nil {
			return err
		}
		if APIUser, found := secretData.Get("", "ICINGA_API_USER"); found {
			writeENVVar("ICINGA_API_USER", APIUser)
		}

		if Password, found := secretData.Get("", "ICINGA_API_PASSWORD"); found {
			writeENVVar("ICINGA_API_PASS", Password)
		}
	}

	return nil
}

func writeENVVar(key, value string) {
	f, err := os.OpenFile(ENVFILE, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println("FILE ERROR****")
	}
	f.WriteString(key + "=" + value + "\n")
	f.Close()
}
func (c *Cluster) runNodeExecutor(podName, podIp, namespace, containerName string) error {
	eo := ExecOptions{
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
		Command: []string{
			"apt-get update",
			"apt-get install wget -y",
			"wget http://" + podIp + ":80",
		},
		Executor: &RemoteBashExecutor{},
		Client:   c.Kube.Client,
		Config:   c.Kube.Config,
	}
	retry := 5
	for retry > 0 {
		resp, err := eo.Run(2)
		fmt.Println(resp)
		if err == nil && strings.Contains(resp, "HTTP request sent, awaiting response... 200 OK") {
			term.Successln("Network is ok from", podName, "to", podIp)
			return nil
		}
		time.Sleep(5 * time.Second)
		retry--
	}
	return errors.New("Network is not ok from", podName, "to", podIp).Err()
}

func (c *Cluster) runMasterExecutor(masterNode, podIp string) error {
	req := appscodeSSH.SSHGetRequest{
		Namespace:    c.Namespace,
		ClusterName:  c.Name,
		InstanceName: masterNode,
	}

	retry := 5

	for retry > 0 {
		resp, err := c.callClusterSSHApi(req)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		stdIn := newStringReader([]string{
			"wget http://" + podIp + ":80",
		})
		DefaultWriter.Flush()
		var output string = ""
		if resp.Command != "" {
			arg := str.ToArgv(resp.Command)
			name, arg := arg[0], arg[1:]
			arg = append(arg, "--command", "wget http://"+podIp+":80")
			cmd := exec.Command(name, arg...)
			cmd.Stdin = stdIn
			cmd.Stdout = DefaultWriter
			cmd.Stderr = DefaultWriter
			err = cmd.Run()
			output = DefaultWriter.Output()

		} else {
			keySigner, _ := ssh.ParsePrivateKey(resp.SshKey.PrivateKey)
			config := &ssh.ClientConfig{
				User: resp.User,
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(keySigner),
				},
			}
			conn, _ := ssh.Dial("tcp", fmt.Sprintf("%v:%v", resp.InstanceAddr, resp.InstancePort), config)
			defer conn.Close()
			session, _ := conn.NewSession()
			session.Stdout = DefaultWriter
			session.Stderr = DefaultWriter
			session.Stdin = os.Stdin
			session.Run("wget http://" + podIp + ":80")
			output = DefaultWriter.Output()
			session.Close()

		}
		fmt.Println(output, err)
		if err == nil && strings.Contains(output, "HTTP request sent, awaiting response... 200 OK") {
			term.Successln("Network is ok from master to ", podIp)
			return nil
		}
		time.Sleep(5 * time.Second)
		retry--
	}
	return errors.New("Can't connect with master", DefaultWriter.Output()).Err()
}

func (c *Cluster) InstallNginxService(name string) (string, error) {
	fmt.Println("Installing nginx service ", name)
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: defaultNamespace,
			Labels: map[string]string{
				"app": name,
			},
		},

		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Ports: []apiv1.ServicePort{
				{
					Port:       80,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(80),
				},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}
	if _, err := c.Kube.Client.CoreV1().Services(defaultNamespace).Create(svc); err != nil {
		return "", errors.FromErr(err).Err()
	}
	time.Sleep(10 * time.Second)
	var service *apiv1.Service
	err := backoff.Retry(func() error {
		var err error
		service, err = c.Kube.Client.CoreV1().Services(defaultNamespace).Get(name, metav1.GetOptions{})
		return err
	}, NewExponentialBackOff())

	if err != nil {
		return "", errors.FromErr(err).Err()
	}
	return service.Spec.ClusterIP, nil
}

func (c *Cluster) InstallNginxPod(name, node string) ([]apiv1.Pod, error) {
	dep := new(extensions.Deployment)
	dep.Name = name
	container := apiv1.Container{
		Name:  name,
		Image: "nginx",
		Ports: []apiv1.ContainerPort{
			{
				ContainerPort: 80,
				Protocol:      "TCP",
			},
		},
	}

	dep.Spec.Template.Labels = map[string]string{
		"app": name,
	}
	dep.Spec.Replicas = types.Int32P(1)
	dep.Spec.Template.Spec.Containers = []apiv1.Container{container}
	dep.Spec.Template.Spec.NodeSelector = map[string]string{
		"kubernetes.io/hostname": node,
	}

	if _, err := c.Kube.Client.ExtensionsV1beta1().Deployments(defaultNamespace).Create(dep); err != nil {
		return nil, errors.FromErr(err).Err()
	}

	fmt.Println("Sleeping for 10 second...")
	waitTime(10)

	labelMap := map[string]string{
		"app": name,
	}

	retry := 10
	for retry > 0 {
		pods, err := c.Kube.Client.CoreV1().Pods(defaultNamespace).List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labelMap).String(),
		})
		if err != nil {
			return nil, errors.FromErr(err).Err()
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase == apiv1.PodRunning {
				log.Infoln(pod.Name, " is running")
				return pods.Items, nil
			}
		}
		time.Sleep(30 * time.Second)
		retry--
	}

	return nil, errors.New("No pod running").Err()

}

func (c *Cluster) DeleteNginxPod(name string) error {
	fmt.Println("Deleting nginx pod", name)
	trueVar := true
	dep, err := c.Kube.Client.ExtensionsV1beta1().Deployments(defaultNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		errors.FromErr(err).Err()
	}
	dep.Spec.Replicas = types.Int32P(0)
	c.Kube.Client.ExtensionsV1beta1().Deployments(defaultNamespace).Update(dep)
	time.Sleep(5 * time.Second)
	return c.Kube.Client.ExtensionsV1beta1().Deployments(defaultNamespace).Delete(name, &metav1.DeleteOptions{
		OrphanDependents: &trueVar,
	})
}

func (c *Cluster) DeleteNginxService(name string) error {
	return c.Kube.Client.CoreV1().Services(defaultNamespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *Cluster) InstallKubeConfig() error {
	req := proto.ClusterClientConfigRequest{
		Name: c.Name,
	}
	log.Infoln("Looking for cluster context")
	var resp *proto.ClusterClientConfigResponse
	var err error
	for retry := 10; retry > 0; retry = retry - 1 {
		resp, err = c.callClusterConfigApi(req)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Minute)
		log.Infoln("Retrying ...")
	}
	if resp == nil {
		return errors.New("Cluster configuratoin not found").Err()
	}
	konfig := &KubeConfig{
		APIVersion: "v1",
		Kind:       "Config",
		Preferences: map[string]interface{}{
			"colors": true,
		},
		Clusters: []*ClustersInfo{
			{
				Name: resp.ClusterDomain,
				Cluster: map[string]interface{}{
					"certificate-authority-data": resp.CaCert,
					"server":                     resp.ApiServerUrl,
				},
			},
		},
		Contexts: []*ContextInfo{
			{
				Name: resp.ContextName,
				Contextt: map[string]interface{}{
					"cluster": resp.ClusterDomain,
					"user":    resp.ClusterUserName,
				},
			},
		},
		CurrentContext: resp.ContextName,
	}
	konfig.Users = append(konfig.Users, setUser(&UserInfo{}, resp))

	output, err := yaml.Marshal(konfig)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if _, err := os.Stat(comissionarKubeConfigPath); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(comissionarKubeConfigPath), 0755)
		fmt.Println(err)
	}
	if err := ioutil.WriteFile(comissionarKubeConfigPath, output, 0755); err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func setUser(u *UserInfo, resp *proto.ClusterClientConfigResponse) *UserInfo {
	u.Name = resp.ClusterUserName
	if resp.UserToken != "" {
		u.User = map[string]interface{}{
			"token": resp.UserToken,
		}
	} else if resp.Password != "" {
		u.User = map[string]interface{}{
			"username": resp.ClusterUserName,
			"password": resp.Password,
		}
	} else {
		u.User = map[string]interface{}{
			"client-certificate-data": resp.UserCert,
			"client-key-data":         resp.UserKey,
		}
	}
	return u
}

type ClustersInfo struct {
	Name    string                 `json:"name"`
	Cluster map[string]interface{} `json:"cluster"`
}

type UserInfo struct {
	Name string                 `json:"name"`
	User map[string]interface{} `json:"user"`
}

type ContextInfo struct {
	Name     string                 `json:"name"`
	Contextt map[string]interface{} `json:"context"`
}

// Adapted from https://github.com/kubernetes/client-go/blob/master/tools/clientcmd/api/v1/types.go#L27
// Simplified to avoid dependency on client-go
type KubeConfig struct {
	Kind           string                 `json:"kind,omitempty"`
	APIVersion     string                 `json:"apiVersion,omitempty"`
	Clusters       []*ClustersInfo        `json:"clusters"`
	Contexts       []*ContextInfo         `json:"contexts"`
	CurrentContext string                 `json:"current-context"`
	Preferences    map[string]interface{} `json:"preferences"`
	Users          []*UserInfo            `json:"users"`
	Extensions     json.RawMessage        `json:"extensions,omitempty"`
}

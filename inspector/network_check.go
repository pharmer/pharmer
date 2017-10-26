package inspector

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
	. "github.com/appscode/pharmer/cloud"
	"k8s.io/apimachinery/pkg/util/intstr"
	"github.com/appscode/go/wait"
	"k8s.io/apimachinery/pkg/labels"
	"github.com/mgutz/str"
	"github.com/appscode/go-term"
	"golang.org/x/crypto/ssh"
)

var comissionarKubeConfigPath = clientcmd.RecommendedHomeFile

const (
	defaultNamespace = "default"
	AppscodeIcinga   = "appscode-icinga"
	ENV              = ".env"
	ENVFILE          = "/srv/appscode/.env"
	Server			 = "pingserver"
)



func (i *Inspector) getNodes() (*apiv1.NodeList, error) {
	nodes := &apiv1.NodeList{}
	if err := i.client.CoreV1().RESTClient().Get().Resource("nodes").Do().Into(nodes); err != nil {
		return nodes, errors.FromErr(err).Err()
	}
	return nodes, nil
}


func (i *Inspector) runNodeExecutor(podName, podIp, namespace, containerName string) error {
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
		Client:   i.client,
		Config:   i.config,
	}
	//s := i.cm.
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

func (i *Inspector) runMasterExecutor(masterNode apiv1.Node, podIp string) error {
	retry := 5
	command := ""
	for retry > 0 {
		var err error
		stdIn := newStringReader([]string{
			"wget http://" + podIp + ":80",
		})
		DefaultWriter.Flush()
		var output string = ""
		if command != "" {
			arg := str.ToArgv(command)
			name, arg := arg[0], arg[1:]
			arg = append(arg, "--command", "wget http://"+podIp+":80")
			cmd := exec.Command(name, arg...)
			cmd.Stdin = stdIn
			cmd.Stdout = DefaultWriter
			cmd.Stderr = DefaultWriter
			err = cmd.Run()
			output = DefaultWriter.Output()

		} else {
			keySigner, _ := ssh.ParsePrivateKey(SSHKey(i.ctx).PrivateKey)
			config := &ssh.ClientConfig{
				User: "root",
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(keySigner),
				},
			}
			conn, _ := ssh.Dial("tcp", fmt.Sprintf("%v:%v", masterNode.Status.Addresses[0].Address, 22), config)
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


func (i *Inspector) InstallNginxService() (string, error) {
	fmt.Println("Installing nginx service ", Server)
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Server,
			Namespace: defaultNamespace,
			Labels: map[string]string{
				"app": Server,
			},
		},

		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Ports: []apiv1.ServicePort{
				{
					Port:       80,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: map[string]string{
				"app": Server,
			},
		},
	}
	if _, err := i.client.CoreV1().Services(defaultNamespace).Create(svc); err != nil {
		return "", errors.FromErr(err).Err()
	}
	var service *apiv1.Service
	//attempt := 0
	wait.PollImmediate(RetryInterval, RetryTimeout, func()(bool, error) {
		var err error
		service, err = i.client.CoreV1().Services(defaultNamespace).Get(Server, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return service.Spec.ClusterIP, nil
}

func (i *Inspector) InstallNginx() ([]apiv1.Pod, error)  {
	daemonset := new(extensions.DaemonSet)
	daemonset.Name = Server
	container := apiv1.Container{
		Name:  Server,
		Image: "nginx",
		Ports: []apiv1.ContainerPort{
			{
				ContainerPort: 80,
				Protocol:      "TCP",
			},
		},
		ImagePullPolicy: apiv1.PullIfNotPresent,
	}
	daemonset.Spec.Template.Labels = map[string]string{
		"app": Server,
	}
	daemonset.Spec.Template.Spec.Containers = []apiv1.Container{container}
	if _, err := i.client.ExtensionsV1beta1().DaemonSets(defaultNamespace).Create(daemonset); err != nil {
		return nil, err
	}
	var pods *apiv1.PodList
	attempt := 0
	err := wait.Poll(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		var err error
		pods, err = i.client.CoreV1().Pods(defaultNamespace).List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": Server,
			}).String(),
		})
		Logger(i.ctx).Infof("Attempt %v: Getting nginx pod ...", attempt)
		if err != nil {
			return false, err
		}
		if len(pods.Items) ==0 {
			return false, nil
		}

		for _, item := range pods.Items {
			if item.Status.Phase != "Running" {
				return false, nil
			}
		}
		return true, nil
	})

	return pods.Items, err
}

func (i *Inspector) DeleteNginx() error {
	err := i.client.ExtensionsV1beta1().DaemonSets(defaultNamespace).Delete(Server, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (i *Inspector) CheckDNSPod() error  {
	attempt := 0
	return wait.Poll(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		pods, err := i.client.CoreV1().Pods(metav1.NamespaceSystem).List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"k8s-app": "kube-dns",
			}).String(),
		})
		Logger(i.ctx).Infof("Attempt %v: Getting DNS pod ...", attempt)
		if err != nil {
			return false, err
		}
		for _, item := range pods.Items {
			fmt.Println(item.Name, "  ", item.Status.Phase, "**")
			if item.Status.Phase != "Running" {
				return false, nil
			}
		}
		return true, nil
	})
}


/*
func (i *Inspector) InstallNginxPod(name, node string) ([]apiv1.Pod, error) {
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

	if _, err := i.client.ExtensionsV1beta1().Deployments(defaultNamespace).Create(dep); err != nil {
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


func (i *Inspector) DeleteNginxService(name string) error {
	return c.Kube.Client.CoreV1().Services(defaultNamespace).Delete(name, &metav1.DeleteOptions{})
}

func (i *Inspector) InstallKubeConfig() error {
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
*/

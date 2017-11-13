package inspector

import (
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go-term"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/wait"
	. "github.com/appscode/pharmer/cloud"
	"golang.org/x/crypto/ssh"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultNamespace = "default"
	ENV              = ".env"
	Server           = "pingserver"
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
			fmt.Sprintf(`curl -o -I -L -s -w "%%{http_code}\n" http://%v:80`, podIp),
		},
		Executor: &RemoteBashExecutor{},
		Client:   i.client,
		Config:   i.config,
	}

	retry := 5
	for retry > 0 {
		resp, err := eo.Run(2)
		if err == nil && strings.Contains(resp, "200") {
			term.Successln("Network is ok from", podName, "to", podIp)
			return nil
		}
		time.Sleep(5 * time.Second)
		retry--
	}
	return errors.New("Network is not ok from", podName, "to", podIp).Err()
}

func (i *Inspector) runMasterExecutor(masterNode apiv1.Node, podIp string) error {
	sshCfg, err := GetSSHConfig(i.ctx, i.cluster, masterNode.Name)
	if err != nil {
		return err
	}

	command := fmt.Sprintf(`curl -o -I -L -s -w "%%{http_code}\n" http://%v:80`, podIp)
	keySigner, _ := ssh.ParsePrivateKey(sshCfg.PrivateKey)
	config := &ssh.ClientConfig{
		User: sshCfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keySigner),
		},
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		DefaultWriter.Flush()
		resp, err := ExecuteTCPCommand(command, fmt.Sprintf("%v:%v", sshCfg.HostIP, sshCfg.HostPort), config)
		if err == nil && strings.Contains(resp, "200") {
			term.Successln("Network is ok from master to ", podIp)
			return true, nil
		}
		return false, err
	})
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
					TargetPort: intstr.FromInt(80),
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
	wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		var err error
		service, err = i.client.CoreV1().Services(defaultNamespace).Get(Server, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return service.Spec.ClusterIP, nil
}

func (i *Inspector) DeleteNginxService() error {
	return i.client.CoreV1().Services(defaultNamespace).Delete(Server, &metav1.DeleteOptions{})
}

func (i *Inspector) InstallNginx() ([]apiv1.Pod, error) {
	daemonset := new(extensions.DaemonSet)
	daemonset.Name = Server
	container := apiv1.Container{
		Name:  Server,
		Image: "appscode/inspector-nginx:alpine",
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
		if len(pods.Items) == 0 {
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

	pods, err := i.client.CoreV1().Pods(defaultNamespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app": Server,
		}).String(),
	})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		if err := i.client.CoreV1().Pods(defaultNamespace).Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (i *Inspector) CheckDNSPod() error {
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
			if item.Status.Phase != "Running" {
				return false, nil
			}
		}
		term.Successln("DNS pod is running")
		return true, nil
	})
}

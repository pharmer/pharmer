package cloud

import (
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
)

func TestInitConfigYAML(t *testing.T) {
	ifg := kubeadmapi.InitConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "InitConfiguration",
		},
		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: map[string]string{
				"test": "rgs",
			},
		},
		LocalAPIEndpoint: kubeadmapi.APIEndpoint{
			AdvertiseAddress: "1.2.3.4",
			BindPort:         int32(443),
		},
	}

	conf := ConvertInitConfigFromV1bet1ToV1alpha3(&ifg)
	fmt.Println(conf.APIEndpoint)

	cb, err := yaml.Marshal(conf)
	fmt.Println(string(cb), err)
}

func TestClusterConfigurationYAML(t *testing.T) {
	cfg := kubeadmapi.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "ClusterConfiguration",
		},
		Networking:        kubeadmapi.Networking{},
		KubernetesVersion: "v1.12.0",
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		APIServer: kubeadmapi.APIServer{
			ControlPlaneComponent: kubeadmapi.ControlPlaneComponent{
				ExtraArgs: map[string]string{},
			},
			CertSANs: []string{},
		},
		ControllerManager: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: map[string]string{},
		},
		Scheduler: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: map[string]string{},
		},
		ClusterName: "test",
	}
	conf := ConvertClusterConfigFromV1beta1ToV1alpha3(&cfg)
	fmt.Println(conf)

	cb, err := yaml.Marshal(conf)
	fmt.Println(string(cb), err)
}

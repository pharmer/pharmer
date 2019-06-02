package cloud

import (
	"fmt"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("init config y a m l", func() {

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
	})
	It("cluster configuration y a m l", func() {

		cfg := kubeadmapi.ClusterConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kubeadm.k8s.io/v1beta1",
				Kind:       "ClusterConfiguration",
			},
			Networking:        kubeadmapi.Networking{},
			KubernetesVersion: "v1.12.0",

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
	})
})

package v1alpha1

import (
	. "github.com/appscode/go/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha2"
)

func Convert_Kubeadm_V1alpha1_To_V1alpha2(in *v1alpha1.MasterConfiguration) *v1alpha2.MasterConfiguration {
	conf := &v1alpha2.MasterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha2",
			Kind:       "MasterConfiguration",
		},
		API: v1alpha2.API{
			AdvertiseAddress: in.API.AdvertiseAddress,
			BindPort:         in.API.BindPort,
		},
		Networking: v1alpha2.Networking{
			ServiceSubnet: in.Networking.ServiceSubnet,
			PodSubnet:     in.Networking.PodSubnet,
			DNSDomain:     in.Networking.DNSDomain,
		},
		AuditPolicyConfiguration: v1alpha2.AuditPolicyConfiguration{
			Path:      "",
			LogDir:    "/var/log/kubernetes/audit",
			LogMaxAge: Int32P(2),
		},
		KubernetesVersion: in.KubernetesVersion,
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		APIServerExtraArgs:         in.APIServerExtraArgs,
		ControllerManagerExtraArgs: in.ControllerManagerExtraArgs,
		SchedulerExtraArgs:         in.SchedulerExtraArgs,
		APIServerCertSANs:          in.APIServerCertSANs,
	}
	if in.CloudProvider != "" {
		conf.APIServerExtraArgs["cloud-provider"] = in.CloudProvider
		conf.APIServerExtraArgs["cloud-config"] = in.APIServerExtraVolumes[0].HostPath
		conf.ControllerManagerExtraArgs["cloud-provider"] = in.CloudProvider
		conf.ControllerManagerExtraArgs["cloud-confi"] = in.ControllerManagerExtraVolumes[0].HostPath
	}
	return conf
}

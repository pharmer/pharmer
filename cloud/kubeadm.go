package cloud

import (
	"fmt"
	mrnd "math/rand"
	"regexp"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
)

func GetKubeadmToken() string {
	return fmt.Sprintf("%s.%s", RandStringRunes(6), RandStringRunes(16))
}

func init() {
	mrnd.Seed(time.Now().UnixNano())
}

// Hexadecimal
var letterRunes = []rune("0123456789abcdef")

var (
	// TokenIDRegexpString defines token's id regular expression pattern
	TokenIDRegexpString = "^([a-z0-9]{6})$"
	// TokenIDRegexp is a compiled regular expression of TokenIDRegexpString
	TokenIDRegexp = regexp.MustCompile(TokenIDRegexpString)
	// TokenRegexpString defines id.secret regular expression pattern
	TokenRegexpString = "^([a-z0-9]{6})\\.([a-z0-9]{16})$"
	// TokenRegexp is a compiled regular expression of TokenRegexpString
	TokenRegexp = regexp.MustCompile(TokenRegexpString)
)

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrnd.Intn(len(letterRunes))]
	}
	return string(b)
}

func ParseToken(s string) (string, string, error) {
	split := TokenRegexp.FindStringSubmatch(s)
	if len(split) != 3 {
		return "", "", errors.Errorf("token [%q] was not of form [%q]", s, TokenRegexpString)
	}
	return split[1], split[2], nil
}

func GetLatestKubeadmVerson() (string, error) {
	return FetchFromURL("https://dl.k8s.io/release/stable.txt")
}

func ConvertInitConfigFromV1bet1ToV1alpha3(conf *v1beta1.InitConfiguration) *v1alpha3.InitConfiguration {
	return &v1alpha3.InitConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha3",
			Kind:       "InitConfiguration",
		},
		NodeRegistration: v1alpha3.NodeRegistrationOptions{
			KubeletExtraArgs: conf.NodeRegistration.KubeletExtraArgs,
		},
		APIEndpoint: v1alpha3.APIEndpoint{
			AdvertiseAddress: conf.LocalAPIEndpoint.AdvertiseAddress,
			BindPort:         conf.LocalAPIEndpoint.BindPort,
		},
	}
}

func ConvertClusterConfigFromV1beta1ToV1alpha3(conf *v1beta1.ClusterConfiguration) *v1alpha3.ClusterConfiguration {
	return &v1alpha3.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha3",
			Kind:       "ClusterConfiguration",
		},
		Networking: v1alpha3.Networking{
			ServiceSubnet: conf.Networking.ServiceSubnet,
			PodSubnet:     conf.Networking.PodSubnet,
			DNSDomain:     conf.Networking.DNSDomain,
		},
		KubernetesVersion: conf.KubernetesVersion,
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		APIServerExtraArgs:         conf.APIServer.ExtraArgs,
		ControllerManagerExtraArgs: conf.ControllerManager.ExtraArgs,
		SchedulerExtraArgs:         conf.Scheduler.ExtraArgs,
		APIServerCertSANs:          conf.APIServer.CertSANs,
		ClusterName:                conf.ClusterName,
	}
}

func ConvertJoinConfigFromV1beta1ToV1alpha3(conf *v1beta1.JoinConfiguration) *v1alpha3.JoinConfiguration {
	return &v1alpha3.JoinConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha3",
			Kind:       "JoinConfiguration",
		},
		NodeRegistration: v1alpha3.NodeRegistrationOptions{
			KubeletExtraArgs: conf.NodeRegistration.KubeletExtraArgs,
		},
		Token:                      conf.Discovery.BootstrapToken.Token,
		APIEndpoint:                v1alpha3.APIEndpoint{},
		DiscoveryTokenAPIServers:   []string{conf.Discovery.BootstrapToken.APIServerEndpoint},
		DiscoveryTokenCACertHashes: conf.Discovery.BootstrapToken.CACertHashes,
	}
}

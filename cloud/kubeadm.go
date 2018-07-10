package cloud

import (
	"fmt"
	mrnd "math/rand"
	"regexp"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha2"
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

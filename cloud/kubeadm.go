package cloud

import (
	"fmt"
	mrnd "math/rand"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	semver "github.com/hashicorp/go-version"
)

var InstanceNotFound = errors.New("Instance not found")
var UnsupportedOperation = errors.New("Unsupported operation")

func GetKubeadmToken() string {
	return fmt.Sprintf("%s.%s", RandStringRunes(6), RandStringRunes(16))
}

func init() {
	mrnd.Seed(time.Now().UnixNano())
}

// Hexidecimal
var letterRunes = []rune("0123456789abcdef")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrnd.Intn(len(letterRunes))]
	}
	return string(b)
}

func BuildRuntimeConfig(cluster *api.Cluster) {
	if cluster.Spec.EnableThirdPartyResource {
		if cluster.Spec.RuntimeConfig == "" {
			cluster.Spec.RuntimeConfig = "extensions/v1beta1=true,extensions/v1beta1/thirdpartyresources=true"
		} else {
			cluster.Spec.RuntimeConfig += ",extensions/v1beta1=true,extensions/v1beta1/thirdpartyresources=true"
		}
	}

	version, err := semver.NewVersion(cluster.Spec.KubernetesVersion)
	if err != nil {
		version, err = semver.NewVersion(cluster.Spec.KubernetesVersion)
		if err != nil {
			return
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		// Enable ScheduledJobs: http://kubernetes.io/docs/user-guide/scheduled-jobs/#prerequisites
		if cluster.Spec.EnableScheduledJobResource {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "batch/v2alpha1"
			} else {
				cluster.Spec.RuntimeConfig += ",batch/v2alpha1"
			}
		}

		// http://kubernetes.io/docs/admin/authentication/
		if cluster.Spec.EnableWebhookTokenAuthentication {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "authentication.k8s.io/v1beta1=true"
			} else {
				cluster.Spec.RuntimeConfig += ",authentication.k8s.io/v1beta1=true"
			}
		}

		// http://kubernetes.io/docs/admin/authorization/
		if cluster.Spec.EnableWebhookTokenAuthorization {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "authorization.k8s.io/v1beta1=true"
			} else {
				cluster.Spec.RuntimeConfig += ",authorization.k8s.io/v1beta1=true"
			}
		}
		if cluster.Spec.EnableRBACAuthorization {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "rbac.authorization.k8s.io/v1alpha1=true"
			} else {
				cluster.Spec.RuntimeConfig += ",rbac.authorization.k8s.io/v1alpha1=true"
			}
		}
	}
}

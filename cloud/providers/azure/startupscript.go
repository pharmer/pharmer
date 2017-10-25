package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-version"
	"gopkg.in/ini.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func GetTemplateData(ctx context.Context, cluster *api.Cluster, token, nodeGroup string, externalProvider bool) TemplateData {
	td := TemplateData{
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		KubeadmVersion:    cluster.Spec.MasterKubeadmVersion,
		KubeadmToken:      token,
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		APIBindPort:       6443,
		ExtraDomains:      cluster.Spec.ClusterExternalDomain,
		NetworkProvider:   cluster.Spec.Networking.NetworkProvider,
		NodeGroupName:     nodeGroup,
		Provider:          cluster.Spec.Cloud.CloudProvider,
		ExternalProvider:  externalProvider,
	}
	if cluster.Spec.MasterKubeadmVersion != "" {
		if v, err := version.NewVersion(cluster.Spec.MasterKubeadmVersion); err == nil && v.Prerelease() != "" {
			td.IsPreReleaseVersion = true
		} else {
			if lv, err := GetLatestKubeadmVerson(); err == nil && lv == cluster.Spec.MasterKubeadmVersion {
				td.KubeadmVersion = ""
			}
		}
	}

	{
		if cluster.Spec.Cloud.GCE != nil {
			td.ConfigurationBucket = fmt.Sprintf(`gsutil cat gs://%v/config.sh > /etc/kubernetes/config.sh
			`, cluster.Status.Cloud.GCE.BucketName)
		} else if cluster.Spec.Cloud.AWS != nil {
			td.ConfigurationBucket = fmt.Sprintf(`apt-get install awscli -y
			aws s3api get-object --bucket %v --key config.sh /etc/kubernetes/config.sh`,
				cluster.Status.Cloud.AWS.BucketName)
		}
	}

	cfg := kubeadmapi.MasterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha1",
			Kind:       "MasterConfiguration",
		},
		API: kubeadmapi.API{
			AdvertiseAddress: cluster.Spec.API.AdvertiseAddress,
			BindPort:         cluster.Spec.API.BindPort,
		},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		CloudProvider:     cluster.Spec.Cloud.CloudProvider,
		// AuthorizationModes:
		//Token: token,
		//	TokenTTL:                   cluster.Spec.TokenTTL,
		APIServerExtraArgs:         map[string]string{},
		ControllerManagerExtraArgs: map[string]string{},
		SchedulerExtraArgs:         map[string]string{},
		APIServerCertSANs:          []string{},
	}
	if externalProvider {
		cfg.CloudProvider = "external"
	}

	{
		if cluster.Spec.Cloud.GCE != nil {
			cfg.APIServerExtraArgs["cloud-config"] = cluster.Spec.Cloud.CloudConfigPath
			td.CloudConfigPath = cluster.Spec.Cloud.CloudConfigPath
			// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/cluster/gce/configure-vm.sh#L846
			cfg := ini.Empty()
			err := cfg.Section("global").ReflectFrom(cluster.Spec.Cloud.GCE.CloudConfig)
			if err != nil {
				panic(err)
			}
			var buf bytes.Buffer
			_, err = cfg.WriteTo(&buf)
			if err != nil {
				panic(err)
			}
			td.CloudConfig = buf.String()
		}
	}
	{
		if cluster.Spec.Cloud.Azure != nil {
			cfg.APIServerExtraArgs["cloud-config"] = cluster.Spec.Cloud.CloudConfigPath
			td.CloudConfigPath = cluster.Spec.Cloud.CloudConfigPath

			data, err := json.MarshalIndent(cluster.Spec.Cloud.Azure.CloudConfig, "", "  ")
			if err != nil {
				panic(err)
			}
			td.CloudConfig = string(data)
		}
	}
	{
		extraDomains := []string{}
		if domain := Extra(ctx).ExternalDomain(cluster.Name); domain != "" {
			extraDomains = append(extraDomains, domain)
		}
		if domain := Extra(ctx).InternalDomain(cluster.Name); domain != "" {
			extraDomains = append(extraDomains, domain)
		}
		td.ExtraDomains = strings.Join(extraDomains, ",")
	}
	cb, err := yaml.Marshal(&cfg)
	if err != nil {
		panic(err)
	}
	td.MasterConfiguration = string(cb)
	return td
}

func RenderStartupScript(ctx context.Context, cluster *api.Cluster, token, role, nodeGroup string, externalProvider bool) (string, error) {
	var buf bytes.Buffer
	if err := StartupScriptTemplate.ExecuteTemplate(&buf, role, GetTemplateData(ctx, cluster, token, nodeGroup, externalProvider)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

package aws

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, token string) TemplateData {
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
		Provider:          cluster.Spec.Cloud.CloudProvider,
		ExternalProvider:  false, // AWS does not use out-of-tree CCM
	}
	if cluster.Spec.Cloud.AWS != nil {
		td.ConfigurationBucket = fmt.Sprintf(`
apt-get install awscli -y
aws s3api get-object --bucket %v --key config.sh /etc/kubernetes/config.sh
`, cluster.Status.Cloud.AWS.BucketName)
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
	{
		td.KubeletExtraArgs = map[string]string{}
		for k, v := range cluster.Spec.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		for k, v := range ng.Spec.Template.Spec.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
			api.NodePoolKey: ng.Name,
			api.RoleNodeKey: "",
		}.String()
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = cluster.Spec.Cloud.CloudProvider // --cloud-config is not needed, since IAM is used.
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup) TemplateData {
	td := newNodeTemplateData(ctx, cluster, ng, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: ng.Name,
	}.String()

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
		KubernetesVersion:          cluster.Spec.KubernetesVersion,
		CloudProvider:              cluster.Spec.Cloud.CloudProvider,
		APIServerExtraArgs:         cluster.Spec.APIServerExtraArgs,
		ControllerManagerExtraArgs: cluster.Spec.ControllerManagerExtraArgs,
		SchedulerExtraArgs:         cluster.Spec.SchedulerExtraArgs,
		APIServerCertSANs:          []string{},
	}
	td.MasterConfiguration = &cfg
	return td
}

func KubeConfigScript(kubeadmToken string) (string, error) {
	var buf bytes.Buffer
	var token = struct {
		Token string
	}{Token: kubeadmToken}
	if err := kubConfigScriptTemplate.ExecuteTemplate(&buf, "config", token); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var (
	kubConfigScriptTemplate = template.Must(template.New("config").Parse(`#!/bin/bash
declare -x KUBEADM_TOKEN={{ .Token }}
`))
)

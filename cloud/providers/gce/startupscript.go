package gce

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/hashicorp/go-version"
	"gopkg.in/ini.v1"
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
		ExternalProvider:  false, // GCE does not use out-of-tree CCM
	}
	if cluster.Spec.Cloud.GCE != nil {
		td.ConfigurationBucket = fmt.Sprintf(`gsutil cat gs://%v/config.sh > /etc/kubernetes/config.sh`, cluster.Status.Cloud.GCE.BucketName)
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
		td.KubeletExtraArgs["node-labels"] = fmt.Sprintf("cloud.appscode.com/pool=%s,node-role.kubernetes.io/node=", ng.Name)
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = cluster.Spec.Cloud.CloudProvider // requires --cloud-config
		if cluster.Spec.Cloud.GCE != nil {
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

			// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L41
			td.KubeletExtraArgs["cloud-config"] = "/etc/kubernetes/cloud-config"

			// Kubeadm will send cloud-config to kube-apiserver and kube-controller-manager
			// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L193
			// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L230
		}
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup) TemplateData {
	td := newNodeTemplateData(ctx, cluster, ng, "")
	td.KubeletExtraArgs["node-labels"] = fmt.Sprintf("cloud.appscode.com/pool=%s", ng.Name)

	if cluster.Spec.MasterKubeadmVersion != "" {
		if v, err := version.NewVersion(cluster.Spec.MasterKubeadmVersion); err == nil && v.Prerelease() != "" {
			td.IsPreReleaseVersion = true
		} else {
			if lv, err := GetLatestKubeadmVerson(); err == nil && lv == cluster.Spec.MasterKubeadmVersion {
				td.KubeadmVersion = ""
			}
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

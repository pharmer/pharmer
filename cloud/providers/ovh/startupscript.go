package ovh

import (
	"bytes"
	"context"

	"github.com/appscode/go/errors"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	ini "gopkg.in/ini.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, token string) TemplateData {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.Cloud.CCMCredentialName)
	if err != nil {
		panic(err)
	}
	td := TemplateData{
		ClusterName:      cluster.Name,
		KubeletVersion:   cluster.Spec.KubeletVersion,
		KubeadmVersion:   cluster.Spec.KubeletVersion,
		KubeadmToken:     token,
		CloudCredential:  cred.Spec.Data,
		CAHash:           pubkeypin.Hash(CACert(ctx)),
		CAKey:            string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:    string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress: cluster.APIServerAddress(),
		NetworkProvider:  cluster.Spec.Networking.NetworkProvider,
		Provider:         cluster.Spec.Cloud.CloudProvider,
		ExternalProvider: true,
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
		td.KubeletExtraArgs["cloud-provider"] = "external"
		if cluster.Spec.Cloud.CCMCredentialName == "" {
			panic(errors.New("no cloud controller manager credential found"))
		}

		cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
		if err != nil {
			panic(err)
		}
		typed := credential.Ovh{CommonSpec: credential.CommonSpec(cred.Spec)}
		if ok, err := typed.IsValid(); !ok {
			panic(err)
		}

		cloudConfig := &api.OVHCloudConfig{
			AuthUrl:  auth_url,
			Username: typed.Username(),
			Password: typed.Password(),
			TenantId: typed.TenantID(),
			Region:   cluster.Spec.Cloud.Region,
		}

		cfg := ini.Empty()
		err = cfg.Section("global").ReflectFrom(cloudConfig)
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
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		CloudProvider:              "",
		APIServerExtraArgs:         cluster.Spec.APIServerExtraArgs,
		ControllerManagerExtraArgs: cluster.Spec.ControllerManagerExtraArgs,
		SchedulerExtraArgs:         cluster.Spec.SchedulerExtraArgs,
		APIServerCertSANs:          cluster.Spec.APIServerCertSANs,
	}
	td.MasterConfiguration = &cfg
	return td
}

var (
	customTemplate = `
{{ define "init-os" }}
# We rely on DNS for a lot, and it's just not worth doing a whole lot of startup work if this isn't ready yet.
# ref: https://github.com/kubernetes/kubernetes/blob/443908193d564736d02efdca4c9ba25caf1e96fb/cluster/gce/configure-vm.sh#L24
ensure_basic_networking() {
  until getent hosts $(hostname -f || echo _error_) &>/dev/null; do
    echo 'Waiting for functional DNS (trying to resolve my own FQDN)...'
    sleep 3
  done
  until getent hosts $(hostname -i || echo _error_) &>/dev/null; do
    echo 'Waiting for functional DNS (trying to resolve my own IP)...'
    sleep 3
  done

  echo "Networking functional on $(hostname) ($(hostname -i))"
}

ensure_basic_networking
{{ end }}

`

// INSTANCE_ID=$(/usr/bin/curl -fsSL --retry 5 http://169.254.169.254/v1/instanceid 2> /dev/null)
// PRIVATE_ADDRESS=$(pre-k vultr private-ip --token={{ index .CloudCredential "token" }} --instance-id=$INSTANCE_ID)
)

func (conn *cloudConnector) renderStartupScript(ng *api.NodeGroup, token string) (string, error) {
	tpl, err := StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	tpl, err = tpl.Parse(customTemplate)
	if err != nil {
		return "", err
	}

	var script bytes.Buffer
	if ng.Role() == api.RoleMaster {
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, conn.cluster, ng)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, conn.cluster, ng, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}

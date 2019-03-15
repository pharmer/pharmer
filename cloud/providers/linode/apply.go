package linode

import (
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error
	var acts []api.Action

	if in.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", cm.cluster.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}
	if cm.cluster.Spec.Cloud.InstanceImage, err = cm.conn.DetectInstanceImage(); err != nil {
		return nil, err
	}
	Logger(cm.ctx).Debugln("Linode instance image", cm.cluster.Spec.Cloud.InstanceImage)

	if cm.cluster.Status.Phase == api.ClusterUpgrading {
		return nil, errors.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cm.cluster.Name)
	}
	if cm.cluster.Status.Phase == api.ClusterReady {
		var kc kubernetes.Interface
		kc, err = cm.GetAdminClient()
		if err != nil {
			return nil, err
		}
		if upgrade, err := NewKubeVersionGetter(kc, cm.cluster).IsUpgradeRequested(); err != nil {
			return nil, err
		} else if upgrade {
			cm.cluster.Status.Phase = api.ClusterUpgrading
			if _, err := Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				return nil, err
			}

			return cm.applyUpgrade(dryRun)
		}
	}

	if cm.cluster.Status.Phase == api.ClusterPending {
		a, err := cm.applyCreate(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, ng := range nodeGroups {
			ng.Spec.Nodes = 0
			_, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).Update(ng)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		a, err := cm.applyScale(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.applyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}
	return acts, nil
}

// Creates network, and creates ready master(s)
func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	// FYI: Linode does not support tagging.

	// -------------------------------------------------------------------ASSETS
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var masterNG *api.NodeGroup
	masterNG, err = FindMasterNodeGroup(nodeGroups)
	if err != nil {
		return
	}
	acts = append(acts, api.Action{
		Action:   api.ActionAdd,
		Resource: "Master startup script",
		Message:  "Startup script will be created/updated for master instance",
	})
	if !dryRun {
		if _, err = cm.conn.createOrUpdateStackScript(masterNG, ""); err != nil {
			return
		}
	}
	if masterNG.Status.Nodes < masterNG.Spec.Nodes {
		Logger(cm.ctx).Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Master instance %s will be created", cm.namer.MasterName()),
		})
		if !dryRun {
			var masterServer *api.NodeInfo
			masterServer, err = cm.conn.CreateInstance(cm.namer.MasterName(), "", masterNG)
			if err != nil {
				return
			}
			if masterServer.PrivateIP != "" {
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
					Type:    core.NodeInternalIP,
					Address: masterServer.PrivateIP,
				})
			}
			if masterServer.PublicIP != "" {
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
					Type:    core.NodeExternalIP,
					Address: masterServer.PublicIP,
				})
			}

			var kc kubernetes.Interface
			kc, err = cm.GetAdminClient()
			if err != nil {
				return
			}
			// wait for nodes to start
			if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
				return
			}

			masterNG.Status.Nodes = 1
			masterNG, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(masterNG)
			if err != nil {
				return
			}

			cm.cluster.Status.Phase = api.ClusterReady
			if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				return
			}
			// need to run ccm
			if err = CreateCredentialSecret(cm.ctx, kc, cm.cluster); err != nil {
				return
			}

			var cred *api.Credential
			cred, err = Store(cm.ctx).Credentials().Get(cm.cluster.Spec.CredentialName)
			if err != nil {
				return
			}

			// linode CCM needs this secret to work
			// ref: https://github.com/linode/linode-cloud-controller-manager/blob/26179d04e0b99bb4125c0ef1b8a8d01673c9383f/hack/deploy/ccm-linode-template.yaml#L1-L10
			if err = CreateCredentialSecretWithData(kc, "ccm-linode", metav1.NamespaceSystem,
				map[string][]byte{
					"apiToken": []byte(cred.Spec.Data["token"]),
					"region":   []byte(cm.cluster.Spec.Cloud.Region),
				},
			); err != nil {
				return
			}

		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "MasterInstance",
			Message:  "master instance(s) already exist",
		})
	}

	return
}

// Scales up/down regular node groups
func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var token string
	var kc kubernetes.Interface
	if cm.cluster.Status.Phase != api.ClusterPending {
		kc, err = cm.GetAdminClient()
		if err != nil {
			return
		}
		if !dryRun {
			if token, err = GetExistingKubeadmToken(kc, kubeadmconsts.DefaultTokenDuration); err != nil {
				return
			}
		}

	}
	for _, ng := range nodeGroups {
		if ng.IsMaster() {
			continue
		}
		igm := NewNodeGroupManager(cm.ctx, ng, cm.conn, kc, cm.cluster, token,
			func() error {
				_, e2 := cm.conn.createOrUpdateStackScript(ng, token)
				return e2
			},
			func() error {
				return cm.conn.deleteStackScript(ng)
			},
		)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}
	return
}

// Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var masterNG *api.NodeGroup
	masterNG, err = FindMasterNodeGroup(nodeGroups)
	if err != nil {
		return
	}
	masterNodes := &core.NodeList{}
	masterNodes, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.NodePoolKey: masterNG.Name,
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		return
	} else if err == nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Will delete master instance with name %v", cm.namer.MasterName()),
		})
		if !dryRun {
			for _, masterInstance := range masterNodes.Items {
				err = cm.conn.DeleteInstanceByProviderID(masterInstance.Spec.ProviderID)
				if err != nil {
					Logger(cm.ctx).Infof("Failed to delete instance %s. Reason: %s", masterInstance.Spec.ProviderID, err)
				}
			}
			if err = cm.conn.deleteStackScript(masterNG); err != nil {
				Logger(cm.ctx).Infof("Failed to delete stack script %s. Reason: %s", masterNG.Name, err)
			}
		}
	}

	// Failed
	cm.cluster.Status.Phase = api.ClusterDeleted
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	Logger(cm.ctx).Infof("Cluster %v deletion is deleted successfully", cm.cluster.Name)
	return
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	var kc kubernetes.Interface
	if kc, err = cm.GetAdminClient(); err != nil {
		return
	}

	upm := NewUpgradeManager(cm.ctx, cm, kc, cm.cluster)
	a, err := upm.Apply(dryRun)
	if err != nil {
		return
	}
	acts = append(acts, a...)
	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}
	return
}

package cloud

import (
	"context"
	"fmt"

	"github.com/appscode/pharmer/api"
	semver "github.com/hashicorp/go-version"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type GenericUpgradeManager struct {
	ctx     context.Context
	ssh     SSHExecutor
	kc      kubernetes.Interface
	cluster *api.Cluster
	version string
}

var _ UpgradeManager = &GenericUpgradeManager{}

func NewUpgradeManager(ctx context.Context, ssh SSHExecutor, kc kubernetes.Interface, cluster *api.Cluster, version string) UpgradeManager {
	return &GenericUpgradeManager{ctx: ctx, ssh: ssh, kc: kc, cluster: cluster, version: version}
}

func (upm *GenericUpgradeManager) Apply(dryRun bool) (acts []api.Action, err error) {
	acts = append(acts, api.Action{
		Action:   api.ActionUpdate,
		Resource: "Master upgrade",
		Message:  fmt.Sprintf("Master instance will be upgraded to %v", upm.version),
	})
	if !dryRun {
		if err = upm.MasterUpgrade(); err != nil {
			return
		}
	}

	var nodeGroups []*api.NodeGroup
	if nodeGroups, err = Store(upm.ctx).NodeGroups(upm.cluster.Name).List(metav1.ListOptions{}); err != nil {
		return
	}
	acts = append(acts, api.Action{
		Action:   api.ActionUpdate,
		Resource: "Node group upgrade",
		Message:  fmt.Sprintf("Node group will be upgraded to %v", upm.version),
	})
	if !dryRun {
		for _, ng := range nodeGroups {
			if ng.IsMaster() {
				continue
			}
			if err = upm.NodeGroupUpgrade(ng); err != nil {
				return
			}
		}
	}
	return
}
func (upm *GenericUpgradeManager) MasterUpgrade() error {
	var masterInstance *apiv1.Node
	var err error
	masterInstances, err := upm.kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"node-role.kubernetes.io/master": "",
		}).String(),
	})
	if err != nil {
		return err
	}
	if len(masterInstances.Items) == 1 {
		masterInstance = &masterInstances.Items[0]
	} else {
		return fmt.Errorf("No master found")
	}

	desireVersion, _ := semver.NewVersion(upm.version)
	currentVersion, _ := semver.NewVersion(masterInstance.Status.NodeInfo.KubeletVersion)

	if isPatch(desireVersion, currentVersion) {
		if _, err = upm.ssh.ExecuteSSHCommand("apt-get upgrade kubelet -y", masterInstance); err != nil {
			return err
		}
	}

	if _, err = upm.ssh.ExecuteSSHCommand(fmt.Sprintf("kubeadm upgrade apply %v -y", upm.version), masterInstance); err != nil {
		return err
	}
	return nil
}

func (upm *GenericUpgradeManager) NodeGroupUpgrade(ng *api.NodeGroup) (err error) {
	nodes := &apiv1.NodeList{}
	if upm.kc != nil {
		nodes, err = upm.kc.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				api.NodeLabelKey_NodeGroup: ng.Spec.Template.Spec.SKU,
			}).String(),
		})
		if err != nil {
			return
		}
	}
	desireVersion, _ := semver.NewVersion(upm.version)
	for _, node := range nodes.Items {
		currentVersion, _ := semver.NewVersion(node.Status.NodeInfo.KubeletVersion)
		if isPatch(desireVersion, currentVersion) {
			if _, err = upm.ssh.ExecuteSSHCommand("apt-get upgrade kubelet -y", &node); err != nil {
				return err
			}
			if _, err = upm.ssh.ExecuteSSHCommand(fmt.Sprintf("systemctl restart kubelet"), &node); err != nil {
				return err
			}
		}
	}
	return nil
}

func isPatch(v1, v2 *semver.Version) bool {
	first := v1.Segments()
	second := v2.Segments()
	if first[0] == second[0] && first[1] == second[1] && first[2] != second[2] {
		return true
	}
	return false
}

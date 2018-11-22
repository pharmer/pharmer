package cloud

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	semver "github.com/appscode/go-version"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

type GenericUpgradeManager struct {
	ctx     context.Context
	ssh     SSHGetter
	kc      kubernetes.Interface
	cluster *api.Cluster

	owner string
}

var _ UpgradeManager = &GenericUpgradeManager{}

func NewUpgradeManager(ctx context.Context, ssh SSHGetter, kc kubernetes.Interface, cluster *api.Cluster, owner string) UpgradeManager {
	return &GenericUpgradeManager{ctx: ctx, ssh: ssh, kc: kc, cluster: cluster, owner: owner}
}

func (upm *GenericUpgradeManager) GetAvailableUpgrades() ([]*api.Upgrade, error) {
	// Collect the upgrades kubeadm can do in this list
	upgrades := make([]*api.Upgrade, 0)
	v := NewKubeVersionGetter(upm.kc, upm.cluster)
	clusterVersionStr, clusterVersion, err := v.ClusterVersion()
	if err != nil {
		return nil, err
	}

	// Get current kubeadm CLI version
	kubeadmVersionStr, kubeadmVersion, err := v.KubeadmVersion()
	if err != nil {
		return nil, err
	}

	// Get and output the current latest stable version
	stableVersionStr, stableVersion, err := v.VersionFromCILabel("stable", "stable version")
	if err != nil {
		fmt.Printf("[upgrade/versions] WARNING: %v\n", err)
		fmt.Println("[upgrade/versions] WARNING: Falling back to current kubeadm version as latest stable version")
		stableVersionStr, stableVersion = kubeadmVersionStr, kubeadmVersion
	}

	// Get the kubelet versions in the cluster
	kubeletVersions, err := v.KubeletVersions()
	if err != nil {
		return nil, err
	}

	dnsType, dnsVersion, err := v.DeployedDNSAddon()
	if err != nil {
		return nil, err
	}
	// Construct a descriptor for the current state of the world
	beforeState := api.ClusterState{
		KubeVersion:     clusterVersionStr,
		DNSType:         dnsType,
		DNSVersion:      dnsVersion,
		KubeadmVersion:  kubeadmVersionStr,
		KubeletVersions: kubeletVersions,
	}

	canDoMinorUpgrade := clusterVersion.LessThan(stableVersion)

	// A patch version doesn't exist if the cluster version is higher than or equal to the current stable version
	// in the case that a user is trying to upgrade from, let's say, v1.8.0-beta.2 to v1.8.0-rc.1 (given we support such upgrades experimentally)
	// a stable-1.8 branch doesn't exist yet. Hence this check.

	if patchVersionBranchExists(clusterVersion, stableVersion) {
		currentBranch := getBranchFromVersion(clusterVersionStr)
		versionLabel := fmt.Sprintf("stable-%s", currentBranch)
		description := fmt.Sprintf("version in the v%s series", currentBranch)

		// Get and output the latest patch version for the cluster branch
		patchVersionStr, patchVersion, err := v.VersionFromCILabel(versionLabel, description)
		if err != nil {
			return nil, err
		}

		// Check if a minor version upgrade is possible when a patch release exists
		// It's only possible if the latest patch version is higher than the current patch version
		// If that's the case, they must be on different branches => a newer minor version can be upgraded to
		canDoMinorUpgrade = minorUpgradePossibleWithPatchRelease(stableVersion, patchVersion)
		// If the cluster version is lower than the newest patch version, we should inform about the possible upgrade
		if patchUpgradePossible(clusterVersion, patchVersion) {

			// The kubeadm version has to be upgraded to the latest patch version
			newKubeadmVer := patchVersionStr
			if kubeadmVersion.AtLeast(patchVersion) {
				// In this case, the kubeadm CLI version is new enough. Don't display an update suggestion for kubeadm by making .NewKubeadmVersion equal .CurrentKubeadmVersion
				newKubeadmVer = kubeadmVersionStr
			}

			upgrades = append(upgrades, &api.Upgrade{
				Description: description,
				Before:      beforeState,
				After: api.ClusterState{
					KubeVersion:    patchVersionStr,
					DNSType:        api.CoreDNS,
					DNSVersion:     kubeadmconstants.GetDNSVersion(api.CoreDNS),
					KubeadmVersion: newKubeadmVer,
					// KubeletVersions is unset here as it is not used anywhere in .After
				},
			})
		}
	}
	if canDoMinorUpgrade {
		upgrades = append(upgrades, &api.Upgrade{
			Description: "stable version",
			Before:      beforeState,
			After: api.ClusterState{
				KubeVersion:    stableVersionStr,
				DNSType:        api.CoreDNS,
				DNSVersion:     kubeadmconstants.GetDNSVersion(api.CoreDNS),
				KubeadmVersion: stableVersionStr,
				// KubeletVersions is unset here as it is not used anywhere in .After
			},
		})
	}

	return upgrades, nil
}

func (upm *GenericUpgradeManager) ExecuteSSHCommand(command string, node *core.Node) (string, error) {
	cfg, err := upm.ssh.GetSSHConfig(upm.cluster, node)
	if err != nil {
		return "", err
	}
	keySigner, err := ssh.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return "", err
	}
	config := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keySigner),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return ExecuteTCPCommand(command, fmt.Sprintf("%v:%v", cfg.HostIP, cfg.HostPort), config)
}

// printAvailableUpgrades prints a UX-friendly overview of what versions are available to upgrade to
// TODO look into columnize or some other formatter when time permits instead of using the tabwriter
func (upm *GenericUpgradeManager) PrintAvailableUpgrades(upgrades []*api.Upgrade) {
	// Return quickly if no upgrades can be made
	if len(upgrades) == 0 {
		fmt.Println("Awesome, you're up-to-date! Enjoy!")
		return
	}
	w := os.Stdout
	// The tab writer writes to the "real" writer w
	tabw := tabwriter.NewWriter(w, 10, 4, 3, ' ', 0)

	// Loop through the upgrade possibilities and output text to the command line
	for _, upgrade := range upgrades {

		if upgrade.CanUpgradeKubelets() {
			fmt.Fprintln(w, "Components that will be upgraded after you've upgraded the control plane:")
			fmt.Fprintln(tabw, "COMPONENT\tCURRENT\tAVAILABLE")
			firstPrinted := false

			// The map is of the form <old-version>:<node-count>. Here all the keys are put into a slice and sorted
			// in order to always get the right order. Then the map value is extracted separately
			for _, oldVersion := range sortedSliceFromStringIntMap(upgrade.Before.KubeletVersions) {
				nodeCount := upgrade.Before.KubeletVersions[oldVersion]
				if !firstPrinted {
					// Output the Kubelet header only on the first version pair
					fmt.Fprintf(tabw, "Kubelet\t%d x %s\t%s\n", nodeCount, oldVersion, upgrade.After.KubeVersion)
					firstPrinted = true
					continue
				}
				fmt.Fprintf(tabw, "\t\t%d x %s\t%s\n", nodeCount, oldVersion, upgrade.After.KubeVersion)
			}
			// We should flush the writer here at this stage; as the columns will now be of the right size, adjusted to the above content
			tabw.Flush()
			fmt.Fprintln(w, "")
		}

		fmt.Fprintf(w, "Upgrade to the latest %s:\n", upgrade.Description)
		fmt.Fprintln(w, "")
		fmt.Fprintln(tabw, "COMPONENT\tCURRENT\tAVAILABLE")
		fmt.Fprintf(tabw, "API Server\t%s\t%s\n", upgrade.Before.KubeVersion, upgrade.After.KubeVersion)
		fmt.Fprintf(tabw, "Controller Manager\t%s\t%s\n", upgrade.Before.KubeVersion, upgrade.After.KubeVersion)
		fmt.Fprintf(tabw, "Scheduler\t%s\t%s\n", upgrade.Before.KubeVersion, upgrade.After.KubeVersion)
		fmt.Fprintf(tabw, "Kube Proxy\t%s\t%s\n", upgrade.Before.KubeVersion, upgrade.After.KubeVersion)
		fmt.Fprintf(tabw, "Core DNS\t%s\t%s\n", upgrade.Before.DNSVersion, upgrade.After.DNSVersion)

		// The tabwriter should be flushed at this stage as we have now put in all the required content for this time. This is required for the tabs' size to be correct.
		tabw.Flush()
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "You can now apply the upgrade by executing the following command:")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "\tpharmer edit cluster %s --kubernetes-version=%s\n", upm.cluster.Name, upgrade.After.KubeVersion)
		fmt.Fprintln(w, "")

		if upgrade.Before.KubeadmVersion != upgrade.After.KubeadmVersion {
			fmt.Fprintf(w, "Note: Before you do can perform this upgrade, you have to update kubeadm to %s\n", upgrade.After.KubeadmVersion)
			fmt.Fprintln(w, "")
		}

		fmt.Fprintln(w, "_____________________________________________________________________")
		fmt.Fprintln(w, "")
	}
}
func (upm *GenericUpgradeManager) Apply(dryRun bool) (acts []api.Action, err error) {
	acts = append(acts, api.Action{
		Action:   api.ActionUpdate,
		Resource: "Master upgrade",
		Message:  fmt.Sprintf("Master instance will be upgraded to %v", upm.cluster.Spec.KubernetesVersion),
	})
	if !dryRun {
		if err = upm.MasterUpgrade(); err != nil {
			return
		}

		desiredVersion, _ := semver.NewVersion(upm.cluster.Spec.KubernetesVersion)
		if err = WaitForReadyMasterVersion(upm.ctx, upm.kc, desiredVersion); err != nil {
			return
		}
		// wait for nodes to start
		if err = WaitForReadyMaster(upm.ctx, upm.kc); err != nil {
			return
		}
	}

	var nodeGroups []*api.NodeGroup
	if nodeGroups, err = Store(upm.ctx).Owner(upm.owner).NodeGroups(upm.cluster.Name).List(metav1.ListOptions{}); err != nil {
		return
	}
	acts = append(acts, api.Action{
		Action:   api.ActionUpdate,
		Resource: "Node group upgrade",
		Message:  fmt.Sprintf("Node group will be upgraded to %v", upm.cluster.Spec.KubernetesVersion),
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
	var masterInstance *core.Node
	var err error
	masterInstances, err := upm.kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleMasterKey: "",
		}).String(),
	})
	if err != nil {
		return err
	}
	if len(masterInstances.Items) == 1 {
		masterInstance = &masterInstances.Items[0]
	} else if len(masterInstances.Items) > 1 {
		return errors.Errorf("multiple master found")
	} else {
		return errors.Errorf("no master found")
	}

	desiredVersion, _ := semver.NewVersion(upm.cluster.Spec.KubernetesVersion)
	currentVersion, _ := semver.NewVersion(masterInstance.Status.NodeInfo.KubeletVersion)

	v11, err := semver.NewVersion("1.11.0")
	if err != nil {
		return err
	}

	// ref: https://stackoverflow.com/a/2831449/244009
	steps := []string{
		`echo "#!/bin/bash" > /usr/bin/pharmer.sh`,
		`echo "set -xeou pipefail" >> /usr/bin/pharmer.sh`,
		`echo "export DEBIAN_FRONTEND=noninteractive" >> /usr/bin/pharmer.sh`,
		`echo "export DEBCONF_NONINTERACTIVE_SEEN=true" >> /usr/bin/pharmer.sh`,
		`echo "" >> /usr/bin/pharmer.sh`,
		`echo "apt-get update" >> /usr/bin/pharmer.sh`,
	}
	if !desiredVersion.Equal(currentVersion) {
		patch := desiredVersion.Clone().ToMutator().ResetPrerelease().ResetMetadata().String()
		minor := desiredVersion.Clone().ToMutator().ResetPrerelease().ResetMetadata().ResetPatch().String()
		cni, found := kubernetesCNIVersions[minor]
		if !found {
			return errors.Errorf("kubernetes-cni version is unknown for Kubernetes version %s", desiredVersion)
		}
		prekVer, found := prekVersions[minor]
		if !found {
			return errors.Errorf("pre-k version is unknown for Kubernetes version %s", desiredVersion)
		}

		// Keep using forked kubeadm 1.8.x for: https://github.com/kubernetes/kubernetes/pull/49840
		if minor == "1.8.0" {
			steps = append(steps, fmt.Sprintf(`echo "apt-get upgrade -y kubelet=%s* kubectl=%s* kubernetes-cni=%s*" >> /usr/bin/pharmer.sh`, patch, patch, cni))
		} else if desiredVersion.LessThan(v11) {
			steps = append(steps, fmt.Sprintf(`echo "apt-get upgrade -y kubelet=%s* kubectl=%s* kubeadm=%s* kubernetes-cni=%s*" >> /usr/bin/pharmer.sh`, patch, patch, patch, cni))
		} else {
			steps = append(steps, []string{
				`echo "curl -sSL https://dl.k8s.io/release/$(curl -sSL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubeadm > /usr/bin/kubeadm" >> /usr/bin/pharmer.sh`,
				`echo "chmod a+rx /usr/bin/kubeadm" >> /usr/bin/pharmer.sh`,
			}...)

		}
		steps = append(steps, fmt.Sprintf(`echo "curl -fsSL --retry 5 -o pre-k https://cdn.appscode.com/binaries/pre-k/%s/pre-k-linux-amd64 && chmod +x pre-k && mv pre-k /usr/bin/" >> /usr/bin/pharmer.sh`, prekVer))
	}

	steps = append(steps,
		fmt.Sprintf(`echo "pre-k check master-status --timeout=-1s --kubeconfig=/etc/kubernetes/admin.conf" >> /usr/bin/pharmer.sh`))
	steps = append(steps,
		fmt.Sprintf(`echo "kubeadm upgrade apply %v -y" >> /usr/bin/pharmer.sh`, upm.cluster.Spec.KubernetesVersion))

	if desiredVersion.Compare(v11) >= 0 {
		steps = append(steps,
			fmt.Sprintf(`echo "kubectl drain %s --ignore-daemonsets" >> /usr/bin/pharmer.sh`, masterInstance.Name),
			fmt.Sprintf(`echo "apt-get upgrade -y kubelet kubeadm" >> /usr/bin/pharmer.sh`),
			fmt.Sprintf(`echo "kubectl uncordon %s" >> /usr/bin/pharmer.sh`, masterInstance.Name))
	}

	steps = append(steps,
		`chmod +x /usr/bin/pharmer.sh`,
		`nohup /usr/bin/pharmer.sh >> /var/log/pharmer.log 2>&1 &`,
	)
	cmd := fmt.Sprintf("sh -c '%s'", strings.Join(steps, "; "))
	Logger(upm.ctx).Infof("Upgrading server %s using `%s`", masterInstance.Name, cmd)

	if _, err = upm.ExecuteSSHCommand(cmd, masterInstance); err != nil {
		return err
	}
	return nil
}

func (upm *GenericUpgradeManager) NodeGroupUpgrade(ng *api.NodeGroup) (err error) {
	nodes := &core.NodeList{}
	if upm.kc != nil {
		nodes, err = upm.kc.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				api.NodePoolKey: ng.Name,
			}).String(),
		})
		if err != nil {
			return
		}
	}
	desiredVersion, _ := semver.NewVersion(upm.cluster.Spec.KubernetesVersion)
	v11, err := semver.NewVersion("1.11.0")
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		currentVersion, _ := semver.NewVersion(node.Status.NodeInfo.KubeletVersion)
		if !desiredVersion.Equal(currentVersion) {
			patch := desiredVersion.Clone().ToMutator().ResetPrerelease().ResetMetadata().String()
			minor := desiredVersion.Clone().ToMutator().ResetPrerelease().ResetMetadata().ResetPatch().String()
			cni, found := kubernetesCNIVersions[minor]
			if !found {
				return errors.Errorf("kubernetes-cni version is unknown for Kubernetes version %s", desiredVersion)
			}
			prekVer, found := prekVersions[minor]
			if !found {
				return errors.Errorf("pre-k version is unknown for Kubernetes version %s", desiredVersion)
			}
			// ref: https://stackoverflow.com/a/2831449/244009
			steps := []string{
				`echo "#!/bin/bash" > /usr/bin/pharmer.sh`,
				`echo "set -xeou pipefail" >> /usr/bin/pharmer.sh`,
				`echo "export DEBIAN_FRONTEND=noninteractive" >> /usr/bin/pharmer.sh`,
				`echo "export DEBCONF_NONINTERACTIVE_SEEN=true" >> /usr/bin/pharmer.sh`,
				`echo "" >> /usr/bin/pharmer.sh`,
				`echo "apt-get update" >> /usr/bin/pharmer.sh`,
			}

			// Keep using forked kubeadm 1.8.x for: https://github.com/kubernetes/kubernetes/pull/49840
			if minor == "1.8.0" {
				steps = append(steps,
					fmt.Sprintf(`echo "apt-get upgrade -y kubelet=%s* kubectl=%s* kubernetes-cni=%s*" >> /usr/bin/pharmer.sh`, patch, patch, cni),
				)
			} else {
				steps = append(steps,
					fmt.Sprintf(`echo "apt-get upgrade -y kubelet=%s* kubectl=%s* kubeadm=%s* kubernetes-cni=%s*" >> /usr/bin/pharmer.sh`, patch, patch, patch, cni),
				)
			}

			if desiredVersion.Compare(v11) >= 0 {
				steps = append(steps,
					fmt.Sprintf(`echo "kubeadm upgrade node config --kubelet-version \$(kubelet --version | cut -d '"'"' '"'"' -f 2)" >> /usr/bin/pharmer.sh`))
			}
			steps = append(steps,
				fmt.Sprintf(`echo "curl -fsSL --retry 5 -o pre-k https://cdn.appscode.com/binaries/pre-k/%s/pre-k-linux-amd64 && chmod +x pre-k && mv pre-k /usr/bin/" >> /usr/bin/pharmer.sh`, prekVer),
				`echo "systemctl restart kubelet" >> /usr/bin/pharmer.sh`,
				`chmod +x /usr/bin/pharmer.sh`,
				`nohup /usr/bin/pharmer.sh >> /var/log/pharmer.log 2>&1 &`,
			)
			cmd := fmt.Sprintf("sh -c '%s'", strings.Join(steps, "; "))
			Logger(upm.ctx).Infof("Upgrading server %s using `%s`", node.Name, cmd)

			if _, err = upm.ExecuteSSHCommand(cmd, &node); err != nil {
				return err
			}
		}
	}
	return nil
}

// sortedSliceFromStringIntMap returns a slice of the keys in the map sorted alphabetically
func sortedSliceFromStringIntMap(strMap map[string]uint32) []string {
	strSlice := []string{}
	for k := range strMap {
		strSlice = append(strSlice, k)
	}
	sort.Strings(strSlice)
	return strSlice
}

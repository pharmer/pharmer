package cloud

import (
	"fmt"
	"strings"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	versionutil "k8s.io/kubernetes/pkg/util/version"
)

// Easy to implement a fake variant of this interface for unit testing
type VersionGetter interface {
	// ClusterVersion should return the version of the cluster i.e. the API Server version
	ClusterVersion() (string, *versionutil.Version, error)
	// MasterKubeadmVersion should return the version of the kubeadm CLI
	KubeadmVersion() (string, *versionutil.Version, error)
	// GetKubeDNSVersion returns the right kube-dns version for a specific k8s version
	KubeDNSVersion() (string, error)
	// VersionFromCILabel should resolve CI labels like `latest`, `stable`, `stable-1.8`, etc. to real versions
	VersionFromCILabel(string, string) (string, *versionutil.Version, error)
	// KubeletVersions should return a map with a version and a number that describes how many kubelets there are for that version
	KubeletVersions() (map[string]uint16, error)
}

// KubeVersionGetter handles the version-fetching mechanism from external sources
type KubeVersionGetter struct {
	client  kubernetes.Interface
	cluster *api.Cluster
}

// NewKubeVersionGetter returns a new instance of KubeVersionGetter
func NewKubeVersionGetter(client kubernetes.Interface, cluster *api.Cluster) VersionGetter {
	return &KubeVersionGetter{
		client:  client,
		cluster: cluster,
	}
}

// ClusterVersion gets API server version
func (g *KubeVersionGetter) ClusterVersion() (string, *versionutil.Version, error) {
	clusterVersionInfo, err := g.client.Discovery().ServerVersion()
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't fetch cluster version from the API Server: %v", err)
	}
	fmt.Println(fmt.Sprintf("[upgrade/versions] Cluster version: %s", clusterVersionInfo.String()))

	clusterVersion, err := versionutil.ParseSemantic(clusterVersionInfo.String())
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't parse cluster version: %v", err)
	}
	return clusterVersionInfo.String(), clusterVersion, nil
}

// MasterKubeadmVersion gets kubeadm version
func (g *KubeVersionGetter) KubeadmVersion() (string, *versionutil.Version, error) {
	kubeadmVersion, err := versionutil.ParseSemantic(g.cluster.Spec.MasterKubeadmVersion)
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't parse kubeadm version: %v", err)
	}
	fmt.Println(fmt.Sprintf("[upgrade/versions] kubeadm version: %s", g.cluster.Spec.MasterKubeadmVersion))

	return g.cluster.Spec.MasterKubeadmVersion, kubeadmVersion, nil
}

//k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/dns/versions.go
// Here we get the value from dns image. originally it was static
func (g *KubeVersionGetter) KubeDNSVersion() (string, error) {
	allDNS, err := g.client.CoreV1().Pods(metav1.NamespaceSystem).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.KubeSystem_App: "kube-dns",
		}).String(),
	})
	if err != nil {
		return "", err
	}
	if len(allDNS.Items) == 0 {
		return "", fmt.Errorf("No DNS pod found")
	}
	dnsImage := allDNS.Items[0].Spec.Containers[0].Image
	imageInfo := strings.Split(dnsImage, ":")
	if len(imageInfo) != 2 {
		return "", fmt.Errorf("Couldn't parse dns version")
	}
	return imageInfo[1], nil
}

// VersionFromCILabel resolves a version label like "latest" or "stable" to an actual version using the public Kubernetes CI uploads
func (g *KubeVersionGetter) VersionFromCILabel(ciVersionLabel, description string) (string, *versionutil.Version, error) {
	versionStr, err := util.KubernetesReleaseVersion(ciVersionLabel)
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't fetch latest %s version from the internet: %v", description, err)
	}

	if description != "" {
		fmt.Println(fmt.Sprintf("[upgrade/versions] Latest %s: %s", description, versionStr))
	}

	ver, err := versionutil.ParseSemantic(versionStr)
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't parse latest %s version: %v", description, err)
	}
	return versionStr, ver, nil
}

// KubeletVersions gets the versions of the kubelets in the cluster
func (g *KubeVersionGetter) KubeletVersions() (map[string]uint16, error) {
	nodes, err := g.client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("couldn't list all nodes in cluster")
	}
	return computeKubeletVersions(nodes.Items), nil
	return nil, fmt.Errorf("couldn't list all nodes in cluster")
}

// computeKubeletVersions returns a string-int map that describes how many nodes are of a specific version
func computeKubeletVersions(nodes []apiv1.Node) map[string]uint16 {
	kubeletVersions := map[string]uint16{}
	for _, node := range nodes {
		kver := node.Status.NodeInfo.KubeletVersion
		if _, found := kubeletVersions[kver]; !found {
			kubeletVersions[kver] = 1
			continue
		}
		kubeletVersions[kver]++
	}
	return kubeletVersions
}

func getBranchFromVersion(version string) string {
	return strings.TrimPrefix(version, "v")[:3]
}

func patchVersionBranchExists(clusterVersion, stableVersion *versionutil.Version) bool {
	return stableVersion.AtLeast(clusterVersion)
}

func patchUpgradePossible(clusterVersion, patchVersion *versionutil.Version) bool {
	return clusterVersion.LessThan(patchVersion)
}

func minorUpgradePossibleWithPatchRelease(stableVersion, patchVersion *versionutil.Version) bool {
	return patchVersion.LessThan(stableVersion)
}

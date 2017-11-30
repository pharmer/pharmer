package cloud

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	versionutil "k8s.io/kubernetes/pkg/util/version"
)

var (
	kubeReleaseBucketURL  = "https://dl.k8s.io"
	kubeReleaseRegex      = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)([-0-9a-zA-Z_\.+]*)?$`)
	kubeReleaseLabelRegex = regexp.MustCompile(`^[[:lower:]]+(-[-\w_\.]+)?$`)
	kubeBucketPrefixes    = regexp.MustCompile(`^((release|ci|ci-cross)/)?([-\w_\.+]+)$`)
)

// KubernetesReleaseVersion is helper function that can fetch
// available version information from release servers based on
// label names, like "stable" or "latest".
//
// If argument is already semantic version string, it
// will return same string.
//
// In case of labels, it tries to fetch from release
// servers and then return actual semantic version.
//
// Available names on release servers:
//  stable      (latest stable release)
//  stable-1    (latest stable release in 1.x)
//  stable-1.0  (and similarly 1.1, 1.2, 1.3, ...)
//  latest      (latest release, including alpha/beta)
//  latest-1    (latest release in 1.x, including alpha/beta)
//  latest-1.0  (and similarly 1.1, 1.2, 1.3, ...)
func KubernetesReleaseVersion(version string) (string, error) {
	if kubeReleaseRegex.MatchString(version) {
		if strings.HasPrefix(version, "v") {
			return version, nil
		}
		return "v" + version, nil
	}

	bucketURL, versionLabel, err := splitVersion(version)
	if err != nil {
		return "", err
	}
	if kubeReleaseLabelRegex.MatchString(versionLabel) {
		url := fmt.Sprintf("%s/%s.txt", bucketURL, versionLabel)
		body, err := FetchFromURL(url)
		if err != nil {
			return "", err
		}
		// Re-validate received version and return.
		return KubernetesReleaseVersion(body)
	}
	return "", fmt.Errorf("version %q doesn't match patterns for neither semantic version nor labels (stable, latest, ...)", version)
}

// Internal helper: split version parts,
// Return base URL and cleaned-up version
func splitVersion(version string) (string, string, error) {
	var urlSuffix string
	subs := kubeBucketPrefixes.FindAllStringSubmatch(version, 1)
	if len(subs) != 1 || len(subs[0]) != 4 {
		return "", "", fmt.Errorf("invalid version %q", version)
	}

	switch {
	case strings.HasPrefix(subs[0][2], "ci"):
		// Special case. CI images populated only by ci-cross area
		urlSuffix = "ci-cross"
	default:
		urlSuffix = "release"
	}
	url := fmt.Sprintf("%s/%s", kubeReleaseBucketURL, urlSuffix)
	return url, subs[0][3], nil
}

// Internal helper: return content of URL
func FetchFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("unable to get URL %q: %s", url, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to fetch file. URL: %q Status: %v", url, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read content of URL %q: %s", url, err.Error())
	}
	return strings.TrimSpace(string(body)), nil
}

// Easy to implement a fake variant of this interface for unit testing
type VersionGetter interface {
	// IsUpgradeRequested returns true if cluster.spec.kubernetesVersion is different from version reported from cluster.
	IsUpgradeRequested() (bool, error)
	// ClusterVersion should return the version of the cluster i.e. the API Server version
	ClusterVersion() (string, *versionutil.Version, error)
	// MasterKubeadmVersion should return the version of the kubeadm CLI
	KubeadmVersion() (string, *versionutil.Version, error)
	// GetKubeDNSVersion returns the right kube-dns version for a specific k8s version
	KubeDNSVersion() (string, error)
	// VersionFromCILabel should resolve CI labels like `latest`, `stable`, `stable-1.8`, etc. to real versions
	VersionFromCILabel(string, string) (string, *versionutil.Version, error)
	// KubeletVersions should return a map with a version and a number that describes how many kubelets there are for that version
	KubeletVersions() (map[string]uint32, error)
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

// IsUpgradeRequested returns true if cluster.spec.kubernetesVersion is different from version reported from cluster.
func (g *KubeVersionGetter) IsUpgradeRequested() (bool, error) {
	if g.cluster.Status.Phase == api.ClusterReady {
		cur, _, err := g.ClusterVersion()
		if err != nil {
			return false, err
		}
		return cur != g.cluster.Spec.KubernetesVersion, nil
	}
	return false, nil
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
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      api.RoleMasterKey,
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	})
	if err != nil {
		return "", nil, err
	}
	nodes, err := g.client.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return "", nil, fmt.Errorf("couldn't list master instances in cluster, Reason: %s", err)
	}
	if len(nodes.Items) == 0 {
		return "", nil, fmt.Errorf("couldn't list master instances in cluster")
	}
	verStr, found := nodes.Items[0].Annotations[api.KubeadmVersionKey]
	if !found {
		return "", nil, fmt.Errorf("master instance %s is missing annotation %s", nodes.Items[0].Name, api.KubeadmVersionKey)
	}
	kubeadmVersion, err := versionutil.ParseSemantic(verStr)
	if err != nil {
		return "", nil, fmt.Errorf("couldn't parse kubeadm version: %v", err)
	}
	fmt.Println(fmt.Sprintf("[upgrade/versions] kubeadm version: %s", verStr))

	return verStr, kubeadmVersion, nil
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
	versionStr, err := KubernetesReleaseVersion(ciVersionLabel)
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
func (g *KubeVersionGetter) KubeletVersions() (map[string]uint32, error) {
	nodes, err := g.client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("couldn't list all nodes in cluster")
	}
	return computeKubeletVersions(nodes.Items), nil
}

// computeKubeletVersions returns a string-int map that describes how many nodes are of a specific version
func computeKubeletVersions(nodes []core.Node) map[string]uint32 {
	kubeletVersions := map[string]uint32{}
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

package cloud

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	versionutil "k8s.io/kubernetes/pkg/util/version"
)

// Easy to implement a fake variant of this interface for unit testing
type VersionGetter interface {
	// ClusterVersion should return the version of the cluster i.e. the API Server version
	ClusterVersion() (string, *versionutil.Version, error)
	// KubeadmVersion should return the version of the kubeadm CLI
	KubeadmVersion() (string, *versionutil.Version, error)
	// VersionFromCILabel should resolve CI labels like `latest`, `stable`, `stable-1.8`, etc. to real versions
	VersionFromCILabel(string, string) (string, *versionutil.Version, error)
	// KubeletVersions should return a map with a version and a number that describes how many kubelets there are for that version
	KubeletVersions() (map[string]uint16, error)
}

// KubeVersionGetter handles the version-fetching mechanism from external sources
type KubeVersionGetter struct {
	client kubernetes.Interface
}

// NewKubeVersionGetter returns a new instance of KubeVersionGetter
func NewKubeVersionGetter(client kubernetes.Interface) VersionGetter {
	return &KubeVersionGetter{
		client: client,
	}
}

// ClusterVersion gets API server version
func (g *KubeVersionGetter) ClusterVersion() (string, *versionutil.Version, error) {
	clusterVersionInfo, err := g.client.Discovery().ServerVersion()
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't fetch cluster version from the API Server: %v", err)
	}
	fmt.Sprintf("[upgrade/versions] Cluster version: %s\n", clusterVersionInfo.String())

	clusterVersion, err := versionutil.ParseSemantic(clusterVersionInfo.String())
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't parse cluster version: %v", err)
	}
	return clusterVersionInfo.String(), clusterVersion, nil
}

// KubeadmVersion gets kubeadm version
func (g *KubeVersionGetter) KubeadmVersion() (string, *versionutil.Version, error) {
	/*kubeadmVersionInfo := version.Get()
	fmt.Sprintf( "[upgrade/versions] kubeadm version: %s\n", kubeadmVersionInfo.String())

	kubeadmVersion, err := versionutil.ParseSemantic(kubeadmVersionInfo.String())
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't parse kubeadm version: %v", err)
	}
	return kubeadmVersionInfo.String(), kubeadmVersion, nil*/
	return "", nil, fmt.Errorf("Couldn't parse kubeadm version: %v", nil)
}

// VersionFromCILabel resolves a version label like "latest" or "stable" to an actual version using the public Kubernetes CI uploads
func (g *KubeVersionGetter) VersionFromCILabel(ciVersionLabel, description string) (string, *versionutil.Version, error) {
	versionStr, err := kubeadmutil.KubernetesReleaseVersion(ciVersionLabel)
	if err != nil {
		return "", nil, fmt.Errorf("Couldn't fetch latest %s version from the internet: %v", description, err)
	}
	fmt.Println(versionStr)
	/*
		if description != "" {
			fmt.Sprintf("[upgrade/versions] Latest %s: %s\n", description, versionStr)
		}

		ver, err := versionutil.ParseSemantic(versionStr)
		if err != nil {
			return "", nil, fmt.Errorf("Couldn't parse latest %s version: %v", description, err)
		}
		return versionStr, ver, nil*/
	return "", nil, fmt.Errorf("Couldn't parse latest %s version: %v", description, nil)
}

// KubeletVersions gets the versions of the kubelets in the cluster
func (g *KubeVersionGetter) KubeletVersions() (map[string]uint16, error) {
	/*nodes, err := g.client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("couldn't list all nodes in cluster")
	}
	return computeKubeletVersions(nodes.Items), nil*/
	return nil, fmt.Errorf("couldn't list all nodes in cluster")
}

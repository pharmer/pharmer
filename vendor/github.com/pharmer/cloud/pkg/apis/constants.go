package apis

import (
	"path/filepath"

	"github.com/appscode/go/runtime"
)

const (
	KeyCloudProvider     = "cloud.pharmer.io/provider"
	KeyClusterCredential = "cloud.pharmer.io/cluster-credential"
	KeyDNSCredential     = "cloud.pharmer.io/dns-credential"
	KeyStorageCredential = "cloud.pharmer.io/storage-credential"
)

const (
	GCE          string = "gce"
	DigitalOcean string = "digitalocean"
	Packet       string = "packet"
	AWS          string = "aws"
	Azure        string = "azure"
	AzureStorage string = "azureStorage"
	Vultr        string = "vultr"
	Linode       string = "linode"
	Scaleway     string = "scaleway"
)

var (
	DataDir string
)

func init() {
	DataDir = filepath.Join(runtime.GOPath(), "src/github.com/pharmer/cloud/data")
}

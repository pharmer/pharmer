package providers

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/cmds/options"
	"github.com/pharmer/pharmer/hack/gendata/providers/aws"
	"github.com/pharmer/pharmer/hack/gendata/providers/azure"
	"github.com/pharmer/pharmer/hack/gendata/providers/digitalocean"
	"github.com/pharmer/pharmer/hack/gendata/providers/gce"
	"github.com/pharmer/pharmer/hack/gendata/providers/linode"
	"github.com/pharmer/pharmer/hack/gendata/providers/packet"
	"github.com/pharmer/pharmer/hack/gendata/providers/scaleway"
	"github.com/pharmer/pharmer/hack/gendata/providers/vultr"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

const (
	Gce          string = "gce"
	DigitalOcean string = "digitalocean"
	Packet       string = "packet"
	Aws          string = "aws"
	Azure        string = "azure"
	Vultr        string = "vultr"
	Linode       string = "linode"
	Scaleway     string = "scaleway"
)

type CloudInterface interface {
	GetName() string
	GetEnvs() []string
	GetCredentials() []data.CredentialFormat
	GetKubernets() []data.Kubernetes
	GetRegions() ([]data.Region, error)
	GetZones() ([]string, error)
	GetInstanceTypes() ([]data.InstanceType, error)
}

func NewCloudProvider(opts *options.CloudData) (CloudInterface, error) {
	switch opts.Provider {
	case Gce:
		return gce.NewGceClient(opts.GCEProjectName, opts.CredentialFile, opts.KubernetesVersions)
		break
	case DigitalOcean:
		return digitalocean.NewDigitalOceanClient(opts.DoToken, opts.KubernetesVersions)
		break
	case Packet:
		return packet.NewPacketClient(opts.PacketToken, opts.KubernetesVersions)
		break
	case Aws:
		return aws.NewAwsClient(opts.AWSRegion, opts.AWSAccessKeyID, opts.AWSSecretAccessKey, opts.KubernetesVersions)
		break
	case Azure:
		return azure.NewAzureClient(opts.AzureTenantId, opts.AzureSubscriptionId, opts.AzureClientId, opts.AzureClientSecret, opts.KubernetesVersions)
		break
	case Vultr:
		return vultr.NewVultrClient(opts.VultrApiKey, opts.KubernetesVersions)
		break
	case Linode:
		return linode.NewLinodeClient(opts.LinodeApiKey, opts.KubernetesVersions)
		break
	case Scaleway:
		return scaleway.NewScalewayClient(opts.ScalewayToken, opts.ScalewayOrganization, opts.KubernetesVersions)
		break
	default:
		return nil, fmt.Errorf("Valid/Supported provider name required")
	}
	return nil, nil
}

func WriteCloudData(cloudInterface CloudInterface) error {
	var err error
	cloudData := data.CloudData{
		Name:        cloudInterface.GetName(),
		Envs:        cloudInterface.GetEnvs(),
		Credentials: cloudInterface.GetCredentials(),
		Kubernetes:  cloudInterface.GetKubernets(),
	}
	cloudData.Regions, err = cloudInterface.GetRegions()
	if err != nil {
		return err
	}
	cloudData.InstanceTypes, err = cloudInterface.GetInstanceTypes()
	if err != nil {
		return err
	}
	dataBytes, err := json.MarshalIndent(cloudData, "", "  ")
	dir, err := GetWriteDir()
	if err != nil {
		return err
	}
	err = util.WriteFile(filepath.Join(dir, cloudData.Name, "cloud.json"), dataBytes)
	return err
}

// wanted directory is [path]/pharmer/data/files
// Current directory is [path]/pharmer/hack/gendata
func GetWriteDir() (string, error) {
	AbsPath, err := filepath.Abs("")
	if err != nil {
		return "", err
	}
	p := strings.TrimSuffix(AbsPath, "/hack/gendata")
	p = filepath.Join(p, "data", "files")
	return p, nil
}

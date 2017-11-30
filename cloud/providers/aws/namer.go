package aws

import (
	"strconv"
	"strings"

	"github.com/appscode/go/crypto/rand"
	stringutil "github.com/appscode/go/strings"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}

func (n namer) VPCName() string {
	return "kubernetes-vpc"
}

func (n namer) DHCPOptionsName() string {
	return n.cluster.Name + "-dhcp-option-set"
}

func (n namer) InternetGatewayName() string {
	return n.cluster.Name + "-internet-gateway"
}

func (n namer) GenMasterSGName() string {
	return n.cluster.Name + "-master-" + rand.Characters(6)
}

func (n namer) GenNodeSGName() string {
	return n.cluster.Name + "-node-" + rand.Characters(6)
}

func (n namer) MasterPDName() string {
	return n.MasterName() + "-pd"
}

// AWS's version of node template
func (n namer) LaunchConfigName(sku string) string {
	return stringutil.DomainForm(n.cluster.Name + "-" + strings.Replace(sku, ".", "-", -1) + "-V" + strconv.FormatInt(n.cluster.Generation, 10))
}

func (n namer) LaunchConfigNameWithContext(sku string, ctxVersion int64) string {
	return stringutil.DomainForm(n.cluster.Name + "-" + strings.Replace(sku, ".", "-", -1) + "-V" + strconv.FormatInt(ctxVersion, 10))
}

// AWS's version of node group
func (n namer) AutoScalingGroupName(sku string) string {
	// return n.ctx.Name + "-node-group-" + sku
	return stringutil.DomainForm(n.cluster.Name + "-" + strings.Replace(sku, ".", "-", -1)) // + "-V" + strconv.FormatInt(n.ctx.ContextVersion, 10))
}

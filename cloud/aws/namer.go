package aws

import (
	"strconv"
	"strings"

	"github.com/appscode/go/crypto/rand"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/pharmer/contexts"
)

type namer struct {
	ctx *contexts.ClusterContext
}

func (n namer) MasterName() string {
	return n.ctx.Name + "-master"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.ctx.Name + "-" + rand.Characters(6)
}

func (n namer) VPCName() string {
	return "kubernetes-vpc"
}

func (n namer) DHCPOptionsName() string {
	return n.ctx.Name + "-dhcp-option-set"
}

func (n namer) InternetGatewayName() string {
	return n.ctx.Name + "-internet-gateway"
}

func (n namer) GenMasterSGName() string {
	return n.ctx.Name + "-master-" + rand.Characters(6)
}

func (n namer) GenNodeSGName() string {
	return n.ctx.Name + "-node-" + rand.Characters(6)
}

func (n namer) MasterPDName() string {
	return n.MasterName() + "-pd"
}

// AWS's versin of node template
func (n namer) LaunchConfigName(sku string) string {
	return stringutil.DomainForm(n.ctx.Name + "-" + strings.Replace(sku, ".", "-", -1) + "-V" + strconv.FormatInt(n.ctx.ContextVersion, 10))
}

func (n namer) LaunchConfigNameWithContext(sku string, ctxVersion int64) string {
	return stringutil.DomainForm(n.ctx.Name + "-" + strings.Replace(sku, ".", "-", -1) + "-V" + strconv.FormatInt(ctxVersion, 10))
}

// AWS's version of node group
func (n namer) AutoScalingGroupName(sku string) string {
	// return n.ctx.Name + "-node-group-" + sku
	return stringutil.DomainForm(n.ctx.Name + "-" + strings.Replace(sku, ".", "-", -1)) // + "-V" + strconv.FormatInt(n.ctx.ContextVersion, 10))
}

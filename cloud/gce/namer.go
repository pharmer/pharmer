package gce

import (
	"strconv"

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

func (n namer) NodePrefix() string {
	return n.ctx.Name + "-node"
}

func (n namer) GenNodeName() string {
	return rand.WithUniqSuffix(n.ctx.Name + "-node")
}

func (n namer) GenSSHKeyExternalID() string {
	return n.ctx.Name + "-" + rand.Characters(6)
}

func (n namer) ReserveIPName() string {
	return n.ctx.Name + "-master-ip"
}

func (n namer) MasterPDName() string {
	return n.MasterName() + "-pd"
}

func (n namer) InstanceTemplateName(sku string) string {
	return stringutil.DomainForm(n.ctx.Name + "-" + sku + "-V" + strconv.FormatInt(n.ctx.ContextVersion, 10))
}

func (n namer) InstanceTemplateNameWithContext(sku string, ctxVersion int64) string {
	return stringutil.DomainForm(n.ctx.Name + "-" + sku + "-V" + strconv.FormatInt(ctxVersion, 10))
}

func (n namer) InstanceGroupName(sku string) string {
	return stringutil.DomainForm(n.ctx.Name + "-" + sku) //+ "-V" + strconv.FormatInt(n.ctx.ContextVersion, 10))
}

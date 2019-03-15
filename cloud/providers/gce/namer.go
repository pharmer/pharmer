package gce

import (
	"strconv"

	"github.com/appscode/go/crypto/rand"
	stringutil "github.com/appscode/go/strings"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

func (n namer) NodePrefix() string {
	return n.cluster.Name + "-node"
}

func (n namer) GenNodeName() string {
	return rand.WithUniqSuffix(n.cluster.Name + "-node")
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}

func (n namer) ReserveIPName() string {
	return n.cluster.Name + "-master-ip"
}

func (n namer) MasterPDName() string {
	return n.MasterName() + "-pd"
}

func (n namer) AdminUsername() string {
	return "pharmer"
}

func (n namer) InstanceTemplateName(sku string) string {
	return stringutil.DomainForm(n.cluster.Name + "-" + sku + "-V" + strconv.FormatInt(n.cluster.Generation, 10))
}

func (n namer) InstanceTemplateNameWithContext(sku string, ctxVersion int64) string {
	return stringutil.DomainForm(n.cluster.Name + "-" + sku + "-V" + strconv.FormatInt(ctxVersion, 10))
}

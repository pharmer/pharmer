/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package gce

import (
	"fmt"
	"strconv"

	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/appscode/go/crypto/rand"
	stringutil "github.com/appscode/go/strings"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
	return n.cluster.Name + "-sshkey"
}

func (n namer) ReserveIPName() string {
	return n.cluster.Name + "-master-ip"
}

func (n namer) MachineDiskName(machine *v1alpha1.Machine) string {
	return machine.Name + "-pd"
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

func (n namer) loadBalancerName() string {
	return fmt.Sprintf("%s-apiserver", n.cluster.Name)
}

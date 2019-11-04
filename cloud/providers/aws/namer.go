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
package aws

import (
	"fmt"
	"strconv"
	"strings"

	api "pharmer.dev/pharmer/apis/v1alpha1"

	stringutil "github.com/appscode/go/strings"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

func (n namer) ControlPlanePolicyName() string {
	return fmt.Sprintf("controller.%s.pharmer", n.cluster.Name)
}

func (n namer) BastionName() string {
	return n.cluster.Name + "-bastion"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-sshkey"
}

func (n namer) VPCName() string {
	return n.cluster.Name + "-vpc"
}

func (n namer) DHCPOptionsName() string {
	return n.cluster.Name + "-dhcp-option-set"
}

func (n namer) InternetGatewayName() string {
	return n.cluster.Name + "-igw"
}

func (n namer) GenMasterSGName() string {
	return n.cluster.Name + "-controlplane"
}

func (n namer) GenNodeSGName() string {
	return n.cluster.Name + "-node" //  + rand.Characters(6)
}

func (n namer) GenBastionSGName() string {
	return n.cluster.Name + "-bastion"
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

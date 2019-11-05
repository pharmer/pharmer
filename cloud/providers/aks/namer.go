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
package aks

import (
	"strings"

	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/appscode/go/crypto/rand"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) ResourceGroupName() string {
	return n.cluster.Name
}

func (n namer) VirtualNetworkName() string {
	return n.cluster.Name + "-vnet"
}

func (n namer) NetworkSecurityGroupName() string {
	return n.cluster.Name + "-nsg"
}

func (n namer) SubnetName() string {
	return n.cluster.Name + "-subnet"
}

func (n namer) RouteTableName() string {
	return n.cluster.Name + "-rt"
}

func (n namer) GenStorageAccountName() string {
	return strings.Replace("k8s-"+rand.WithUniqSuffix(n.cluster.Name), "-", "", -1)
}

func (n namer) AdminUsername() string {
	return "kube"
}

func (n namer) GetNodeGroupName(ng string) string {
	name := strings.ToLower(ng)
	name = strings.Replace(name, "standard", "s", -1)
	name = strings.Replace(name, "pool", "p", -1)
	name = strings.Replace(name, "-", "", -1)
	return name
}

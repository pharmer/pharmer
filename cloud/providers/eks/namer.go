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
package eks

import (
	"fmt"

	api "pharmer.dev/pharmer/apis/v1alpha1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) GetStackServiceRole() string {
	return fmt.Sprintf("EKS-%v-ServiceRole", n.cluster.Name)
}

func (n namer) GetClusterVPC() string {
	return fmt.Sprintf("EKS-%v-VPC", n.cluster.Name)
}

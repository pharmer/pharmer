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
package packet

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/appscode/go/crypto/rand"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

// Deprecated
func (n namer) GenNodeName(ng string) string {
	return rand.WithUniqSuffix(ng)
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-sshkey"
}

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
package providers

import (
	// Imported, so init functions are called and cloud provider gets registered

	_ "pharmer.dev/pharmer/cloud/providers/aks"
	_ "pharmer.dev/pharmer/cloud/providers/aws"
	_ "pharmer.dev/pharmer/cloud/providers/azure"
	_ "pharmer.dev/pharmer/cloud/providers/digitalocean"
	_ "pharmer.dev/pharmer/cloud/providers/dokube"
	_ "pharmer.dev/pharmer/cloud/providers/eks"
	_ "pharmer.dev/pharmer/cloud/providers/gce"
	_ "pharmer.dev/pharmer/cloud/providers/gke"
	_ "pharmer.dev/pharmer/cloud/providers/linode"
	_ "pharmer.dev/pharmer/cloud/providers/packet"
)

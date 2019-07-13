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

package providers

// Imported, so init functions are called and cloud provider gets registered
import (
	_ "github.com/pharmer/pharmer/cloud/providers/aks"
	_ "github.com/pharmer/pharmer/cloud/providers/aws"
	_ "github.com/pharmer/pharmer/cloud/providers/azure"
	_ "github.com/pharmer/pharmer/cloud/providers/digitalocean"
	_ "github.com/pharmer/pharmer/cloud/providers/dokube"
	_ "github.com/pharmer/pharmer/cloud/providers/eks"
	_ "github.com/pharmer/pharmer/cloud/providers/gce"
	_ "github.com/pharmer/pharmer/cloud/providers/gke"
	_ "github.com/pharmer/pharmer/cloud/providers/linode"
	_ "github.com/pharmer/pharmer/cloud/providers/packet"
)

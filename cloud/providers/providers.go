package providers

import (
	_ "github.com/appscode/pharmer/cloud/aws"
	_ "github.com/appscode/pharmer/cloud/azure"
	_ "github.com/appscode/pharmer/cloud/digitalocean"
	_ "github.com/appscode/pharmer/cloud/fake"
	_ "github.com/appscode/pharmer/cloud/gce"
	_ "github.com/appscode/pharmer/cloud/hetzner"
	_ "github.com/appscode/pharmer/cloud/linode"
	_ "github.com/appscode/pharmer/cloud/packet"
	_ "github.com/appscode/pharmer/cloud/scaleway"
	_ "github.com/appscode/pharmer/cloud/softlayer"
	_ "github.com/appscode/pharmer/cloud/vultr"
)

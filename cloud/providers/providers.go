package providers

import (
	_ "github.com/appscode/pharmer/cloud"
	_ "github.com/appscode/pharmer/cloud/providers/aws"
	_ "github.com/appscode/pharmer/cloud/providers/azure"
	_ "github.com/appscode/pharmer/cloud/providers/digitalocean"
	_ "github.com/appscode/pharmer/cloud/providers/fake"
	_ "github.com/appscode/pharmer/cloud/providers/gce"
	_ "github.com/appscode/pharmer/cloud/providers/hetzner"
	_ "github.com/appscode/pharmer/cloud/providers/linode"
	_ "github.com/appscode/pharmer/cloud/providers/packet"
	_ "github.com/appscode/pharmer/cloud/providers/scaleway"
	_ "github.com/appscode/pharmer/cloud/providers/softlayer"
	_ "github.com/appscode/pharmer/cloud/providers/vultr"
	_ "github.com/appscode/pharmer/config"
	_ "github.com/appscode/pharmer/context"
)

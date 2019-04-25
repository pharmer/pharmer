package apis

import (
	"path/filepath"

	"github.com/appscode/go/runtime"
)

const (
	GCE          string = "gce"
	DigitalOcean string = "digitalocean"
	Packet       string = "packet"
	AWS          string = "aws"
	Azure        string = "azure"
	Vultr        string = "vultr"
	Linode       string = "linode"
	Scaleway     string = "scaleway"
)

var (
	DataDir string
)

func init() {
	DataDir = filepath.Join(runtime.GOPath(), "src/github.com/pharmer/cloud/data")
}

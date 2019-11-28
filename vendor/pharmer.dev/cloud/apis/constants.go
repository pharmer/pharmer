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
package apis

import (
	"go/build"
	"path/filepath"
)

const (
	KeyCloudProvider     = "cloud.pharmer.io/provider"
	KeyClusterCredential = "cloud.pharmer.io/cluster-credential"
	KeyDNSCredential     = "cloud.pharmer.io/dns-credential"
	KeyStorageCredential = "cloud.pharmer.io/storage-credential"
)

const (
	GCE          string = "gce"
	DigitalOcean string = "digitalocean"
	Packet       string = "packet"
	AWS          string = "aws"
	Azure        string = "azure"
	AzureStorage string = "azureStorage"
	Vultr        string = "vultr"
	Linode       string = "linode"
	Scaleway     string = "scaleway"
)

var (
	DataDir string
)

func init() {
	DataDir = filepath.Join(build.Default.GOPATH, "src/pharmer.dev/cloud/data")
}

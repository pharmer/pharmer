package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/appscode/go/log"
	"github.com/appscode/go/runtime"
	"github.com/pkg/errors"
)

const (
	docFileName       = "doc.md"
	releaseVersion    = "0.3.1"
	kubernetesVersion = "v1.13.5"
)

type TemplateData struct {
	Provider          CloudProvider
	Release           string
	KubernetesVersion string
}

type CloudProvider struct {
	Capital         string
	Small           string
	URL             string
	Location        string
	ClusterName     string
	MasterNodeCount int
	HASupport       bool
	NodeSpec
}

type NodeSpec struct {
	SKU    string
	CPU    string
	Memory string
}

func getDefaultProviderData() []TemplateData {
	return []TemplateData{
		{
			Provider: CloudProvider{
				Capital:         "AWS",
				Small:           "aws",
				URL:             "https://aws.amazon.com",
				Location:        "us-east-1b",
				ClusterName:     "aws-1",
				MasterNodeCount: 3,
				NodeSpec: NodeSpec{
					SKU:    "t2.medium",
					CPU:    "2",
					Memory: "4 Gb",
				},
				HASupport: true,
			},
			Release:           releaseVersion,
			KubernetesVersion: kubernetesVersion,
		}, {
			Provider: CloudProvider{
				Capital:         "Azure",
				Small:           "azure",
				URL:             "https://azure.microsoft.com",
				Location:        "eastus2",
				ClusterName:     "az1",
				MasterNodeCount: 3,
				NodeSpec: NodeSpec{
					SKU:    "Standard_B2ms",
					CPU:    "2",
					Memory: "4 Gb",
				},
				HASupport: true,
			},
			Release:           releaseVersion,
			KubernetesVersion: kubernetesVersion,
		}, {

			Provider: CloudProvider{
				Capital:         "DigitalOcean",
				Small:           "digitalocean",
				URL:             "https://cloud.digitalocean.com",
				Location:        "nyc1",
				ClusterName:     "d1",
				MasterNodeCount: 3,
				NodeSpec: NodeSpec{
					SKU:    "2gb",
					CPU:    "1",
					Memory: "2 Gb",
				},
				HASupport: true,
			},
			Release:           releaseVersion,
			KubernetesVersion: kubernetesVersion,
		}, {
			Provider: CloudProvider{
				Capital:         "Google Cloud Service",
				Small:           "gce",
				URL:             "https://console.cloud.google.com",
				Location:        "us-central1-f",
				ClusterName:     "g1",
				MasterNodeCount: 3,
				NodeSpec: NodeSpec{
					SKU:    "n1-standard-2",
					CPU:    "2",
					Memory: "7.5 Gb",
				},
				HASupport: true,
			},
			Release:           releaseVersion,
			KubernetesVersion: kubernetesVersion,
		}, {

			Provider: CloudProvider{
				Capital:         "Linode",
				Small:           "linode",
				URL:             "https://linode.com",
				Location:        "us-central",
				ClusterName:     "l1",
				MasterNodeCount: 3,
				NodeSpec: NodeSpec{
					SKU:    "g6-standard-2",
					CPU:    "2",
					Memory: "7.5 Gb",
				},
				HASupport: true,
			},
			Release:           releaseVersion,
			KubernetesVersion: kubernetesVersion,
		}, {

			Provider: CloudProvider{
				Capital:         "Packet",
				Small:           "packet",
				URL:             "https://app.packet.net",
				Location:        "ewr1",
				ClusterName:     "p1",
				MasterNodeCount: 1,
				NodeSpec: NodeSpec{
					SKU:    "baremetal_0",
					CPU:    "4 x86 64bit",
					Memory: "8GB DDR3",
				},
			},
			Release:           releaseVersion,
			KubernetesVersion: kubernetesVersion,
		},
	}
}

// used in template
func (td TemplateData) MachinesetName() string {
	return strings.Replace(strings.ToLower(td.Provider.NodeSpec.SKU), "_", "-", -1) + "-pool"
}

// generate cloud provider docs
func genCloudProviderDocs() error {
	doc, err := ioutil.ReadFile(docFileName)
	if err != nil {
		return errors.Wrap(err, "failed to open template file")
	}

	providerData := getDefaultProviderData()
	for _, data := range providerData {
		tmpl, err := template.New("hugo-template").Parse(string(doc))
		if err != nil {
			return errors.Wrap(err, "failed to create new template")
		}
		provider := data.Provider.Small
		providerFile, err := ioutil.ReadFile(fmt.Sprintf("providers/%s.md", provider))
		if err != nil {
			return errors.Wrapf(err, "failed to open for %s", data.Provider.Small)
		}

		providerTmpl, err := tmpl.Parse(string(providerFile))
		if err != nil {
			return errors.Wrapf(err, "failed to parse file for %s", provider)
		}

		var buf bytes.Buffer
		err = providerTmpl.Execute(&buf, data)
		if err != nil {
			return errors.Wrapf(err, "failed to execute template for %s", provider)
		}

		err = ioutil.WriteFile(
			fmt.Sprintf("%s/src/github.com/pharmer/pharmer/docs/cloud/%s/README.md",
				runtime.GOPath(), provider), buf.Bytes(), 0)
		if err != nil {
			return errors.Wrapf(err, "failed to update doc for %s", provider)
		}
	}

	log.Infoln("successfully updated docs!")
	return nil
}

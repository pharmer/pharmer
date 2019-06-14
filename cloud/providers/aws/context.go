package aws

import (
	"bytes"
	"text/template"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"k8s.io/client-go/kubernetes"
)

type ClusterManager struct {
	*cloud.CloudManager

	conn  *cloudConnector
	namer namer
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "aws"
)

func init() {
	cloud.RegisterCloudManager(UID, func(cluster *api.Cluster, certs *cloud.PharmerCertificates) cloud.Interface {
		return New(cluster, certs)
	})
}

func New(cluster *api.Cluster, certs *cloud.PharmerCertificates) cloud.Interface {
	return &ClusterManager{
		CloudManager: &cloud.CloudManager{
			Cluster: cluster,
			Certs:   certs,
		},
		namer: namer{
			cluster: cluster,
		},
	}
}

// Required for cluster-api controller
func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	err := cloud.CreateNamespace(kc, "aws-provider-system")
	if err != nil {
		return err
	}

	credTemplate := template.Must(template.New("aws-cred").Parse(
		`[default]
aws_access_key_id = {{ .AccessKeyID }}
aws_secret_access_key = {{ .SecretAccessKey }}
region = {{ .Region }}
`))
	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.ClusterConfig().CredentialName)
	if err != nil {
		return err
	}

	data := cred.Spec.Data

	var buf bytes.Buffer
	err = credTemplate.Execute(&buf, struct {
		AccessKeyID     string
		SecretAccessKey string
		Region          string
	}{
		AccessKeyID:     data["accessKeyID"],
		SecretAccessKey: data["secretAccessKey"],
		Region:          cm.Cluster.Spec.Config.Cloud.Region,
	})
	if err != nil {
		return err
	}

	credData := buf.Bytes()

	if err = cloud.CreateSecret(kc, "aws-provider-manager-bootstrap-credentials", "aws-provider-system", map[string][]byte{
		"credentials": credData,
	}); err != nil {
		return err
	}
	return nil
}

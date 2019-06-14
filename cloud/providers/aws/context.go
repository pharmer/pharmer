package aws

import (
	"bytes"
	"text/template"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"k8s.io/client-go/kubernetes"
)

type ClusterManager struct {
	*cloud.CloudManager

	conn  *cloudConnector
	namer namer
}

func (cm *ClusterManager) GetConnector() ClusterApiProviderComponent {
	panic(1)
	return nil
}

// Required for cluster-api controller
func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	err := CreateNamespace(kc, "aws-provider-system")
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

	if err = CreateSecret(kc, "aws-provider-manager-bootstrap-credentials", "aws-provider-system", map[string][]byte{
		"credentials": credData,
	}); err != nil {
		return err
	}
	return nil
}

var _ Interface = &ClusterManager{}

const (
	UID = "aws"
)

func init() {
	RegisterCloudManager(UID, func(cluster *api.Cluster, certs *PharmerCertificates) Interface {
		return New(cluster, certs)
	})
}

func New(cluster *api.Cluster, certs *PharmerCertificates) cloud.Interface {
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

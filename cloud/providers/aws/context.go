package aws

import (
	"bytes"
	"text/template"

	"k8s.io/client-go/kubernetes"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cloud/utils/kube"
)

type ClusterManager struct {
	*cloud.Scope

	conn  *cloudConnector
	namer namer
}

func (cm *ClusterManager) ApplyScale() error {
	panic("implement me")
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "aws"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(s *cloud.Scope) cloud.Interface {
	return &ClusterManager{
		Scope: s,
		namer: namer{
			cluster: s.Cluster,
		},
	}
}

// Required for cluster-api controller
func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	err := kube.CreateNamespace(kc, "aws-provider-system")
	if err != nil {
		return err
	}

	credTemplate := template.Must(template.New("aws-cred").Parse(
		`[default]
aws_access_key_id = {{ .AccessKeyID }}
aws_secret_access_key = {{ .SecretAccessKey }}
region = {{ .Region }}
`))
	cred, err := cm.StoreProvider.Credentials().Get(cm.Cluster.ClusterConfig().CredentialName)
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

	if err = kube.CreateSecret(kc, "aws-provider-manager-bootstrap-credentials", "aws-provider-system", map[string][]byte{
		"credentials": credData,
	}); err != nil {
		return err
	}
	return nil
}

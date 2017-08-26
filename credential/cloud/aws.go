package cloud

import (
	"path/filepath"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/credential"
	"k8s.io/client-go/util/homedir"
)

func IssueAWSCredential(name string) (*api.Credential, error) {
	spec := credential.AWS{}
	err := spec.Load(filepath.Join(homedir.HomeDir(), ".aws", "credentials"))
	if err != nil {
		return nil, err
	}
	return &api.Credential{
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
		Spec: api.CredentialSpec{
			Provider: "AWS",
			Data:     spec.Data,
		},
	}, nil
}

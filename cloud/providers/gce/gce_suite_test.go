package gce

import (
	//go_ctx "context"
	"fmt"
	"testing"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	//. "github.com/appscode/pharmer/cloud/providers/gce"
	//"github.com/appscode/pharmer/config"
	//"github.com/appscode/pharmer/context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//	"time"
	//	"github.com/appscode/pharmer/api"
	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"github.com/appscode/pharmer/phid"
	"context"
	"encoding/json"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
)

func TestGce(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gce Suite")
}

func TestContext(t *testing.T) {
	cfg, err := config.LoadConfig("/home/sanjid/go/src/appscode.com/ark/conf/tigerworks-kube.json")
	fmt.Println(err)
	ctx := cloud.NewContext(context.Background(), cfg)
	cm := New(ctx)

	req := proto.ClusterCreateRequest{
		Name:               "gce-kube",
		Provider:           "gce",
		Zone:               "us-central1-f",
		CredentialUid:      "gce",
		DoNotDelete:        false,
		DefaultAccessLevel: "kubernetes:cluster-admin",
		GceProject:         "tigerworks-kube",
	}
	/*req.NodeSets = make([]*proto.NodeSet, 1)
	req.NodeSets[0] = &proto.NodeSet{
		Sku:   "n1-standard-1",
		Count: int64(1),
	}*/
	cm, err = cloud.GetCloudManager(req.Provider, ctx)
	fmt.Println(err, cm)

	/*cm.cluster = &api.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              req.Name,
			UID:               phid.NewKubeCluster(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.ClusterSpec{
			CredentialName: req.CredentialUid,
		},
	}
	cm.cluster.Spec.Cloud.Zone = req.Zone

	api.AssignTypeKind(cm.cluster)
	if _, err := cloud.Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
		//oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		fmt.Println(err)
	}

	err := cm.DefaultSpec(&req)
	fmt.Println(err)
	fmt.Println(cm.ctx)
	*/ /*cm.Check(&proto.ClusterCreateRequest{
		Name:     "test",
		Provider: "gce",
		Zone:     "us-central1-f",
	})*/ /*
		fmt.Println()*/

}

func TestJson(t *testing.T) {
	data := `{
          "type": "service_account",
          "project_id": "tigerworks-kube",
          "private_key_id": "b19562af11c61ffb92925aaa62ce9f97baf1ea34",
          "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDhkvAvLEeulXAk\nMszTZ48K+hG85ZIsYP9pLY4CX0565ZLafB5KpB8n23tZN08DqImJ08FCATxT042s\nH0pZpRqkGspfK8ElzsEZjEADWaQfVf3ej5WPZYDQbbVOeOA94/qg10QHte68oApL\nqloMLqt0pXrthivQpAmdK4MsfExjy9GK9Kyt6e6YtlmKb7HLeYLRc0GSWiR3x0Hf\nIJW58gjL9k4zT3ZcN/V/6LdbE+7iMSQ2qQQRKGr/JxprzzkeYlkwsKjZJ+0rS/Zt\nuxJ+6SRRg6epbtapL0ZhNRW2+b6+2J6JKicfN/Lg/A8OQd/NmF44TsPDPRq+OhPO\nxzgOaG+PAgMBAAECggEACW6adRkrOiVCK3v69+VUTtzb/FkLvqf4+2M4lT16S5s+\ng7zoNgNyJg7Vs9+jLH+nQ0hubp3HRv4JO3w8y04jCgpGEgADXdsCVK4k8xQ+z5dJ\n5pInFSLlEFILYGefgMGqoNUUzCRoV3dzAWzzF5/6nVsEroU2Cc2sypH03y78sWN9\nPi6Nh8SxCTDfoT/oFAbFK3iuzjfhmlxORwd4c1qQHe6mZqfCGC+ct8HbJg3qYe7K\ne1k1J5bhBIQgzRczA5Gn/+XIP4scPH01R4J1FK9hc1vZ6oW/w/sgLSMDLLJzJ5P6\n5IC89mvwZs+jbhZ2WLdR2X+VQwylHF8yjRuU7204SQKBgQD+ZgZaKlGrRBdJpBo8\nfMfO6qQ6JkY3YrWZOSI23xR663l7dZngCBz/DbtKXFczzDb6Jdjj9ecmH39NF+ct\nnKdgBsRNpkb0Sk6V+SrIEbUOC/T+iAY2gObRtsg1+MVmgci7wt1futpdBKPyCZz3\nmSqEUy3MiGjkrA5KnF5NaVrqawKBgQDi/nYWYmEKE1Rn26pAmBLXKQ6HOqBCMiGC\n+BoCSvei8YL9J/F+C3PYNLdaQd5aIXuS1oRopmJgTVDscnRZ2ExvsTb8EUbR3fV1\nMIw/60663qSuk4FC60JeaExF0KyezHK161wOYDO4C1yLN1mKFXHwnOgIk+2zIG91\ntr3Mr//gbQKBgEyLWCfzCcW1ZChlNvuyM9B/a1CPyZrKmYdz2GaYMqpVhaTvGpB9\nAHSBpjPWmupb7MLRdnQIvjcLTRteMNHZi8bp4lDW0gyY+xJG+WdfZJHIaTvYo73s\nhQber1kF9CdGr6ZHGKLALwnD5qxh1hftvww3ltUuyhjb6CTs7bbvF0rnAoGAHLd6\ncvyBMEgfvn/guwlCIOw1xU/aZGV5Ldt7VtzrFTceji5Wc865GhoZNBbvLVHdE0eG\nOsMJ4QsG+NLF+3PMv7iYryz0W6qL2gaJR7DaJfPyu483pCKlI9JoC9EJdZGB1Zfv\n7nWnNVpim84lyr1Jy9nd1O/5+1ZYI3k568I8ScUCgYEAneRZ1QxsOWNrU4f+rpkm\nsu91U2dist2SiXDAOgcq5U+Kt75t75uTFVPXLQFwNdXis9yezGcmSBj//ScLVEH5\nwBEBM8+WK3VkrO0zWWU4NsTa9cDAtaabdKJohBBPIQcpyT1zZkQAzS9ZZTEm2TYb\nvAr5P1RwHLStsewOFHFM5Ek=\n-----END PRIVATE KEY-----\n",
          "client_email": "dnsservice@tigerworks-kube.iam.gserviceaccount.com",
          "client_id": "112049273811766273667",
          "auth_uri": "https://accounts.google.com/o/oauth2/auth",
          "token_uri": "https://accounts.google.com/o/oauth2/token",
          "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
          "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/dnsservice%40tigerworks-kube.iam.gserviceaccount.com"
        }`
	crd := api.CredentialSpec{
		Data: map[string]string{
			"projectID":      "tigerworks-kube",
			"serviceAccount": data,
		},
	}
	jsn, err := json.Marshal(crd)
	fmt.Println(string(jsn), err)
}

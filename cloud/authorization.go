package cloud

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	proto "github.com/appscode/api/credential/v1beta1"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/credential"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	_iam "github.com/aws/aws-sdk-go/service/iam"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	rupdate "google.golang.org/api/replicapoolupdater/v1beta1"
)

func CheckAuthorization(provider, gceProject string, data map[string]string) (*proto.CredentialIsAuthorizedResponse, error) {
	resp := &proto.CredentialIsAuthorizedResponse{}
	if provider == "aws" {
		resp.Unauthorized, resp.Message = IsAwsUnauthorized(data)
	} else if provider == "gce" {
		resp.Unauthorized, resp.Message = IsGceUnauthorized(gceProject, data)
	} else if provider == "digitalocean" {
		resp.Unauthorized, resp.Message = IsDigitalOceanUnauthorized(data)
	}
	return resp, nil
}

// Returns true if unauthorized
func IsAwsUnauthorized(data map[string]string) (bool, string) {
	var id, secret string
	var found bool
	if id, found = data[credential.AWSAccessKeyID]; !found {
		return true, "Credential missing " + credential.AWSAccessKeyID
	}
	if secret, found = data[credential.AWSSecretAccessKey]; !found {
		return true, "Credential missing " + credential.AWSSecretAccessKey
	}

	defaultRegion := "us-east-1"
	config := &_aws.Config{
		Region:      types.StringP(defaultRegion),
		Credentials: credentials.NewStaticCredentials(id, secret, ""),
	}
	iam := _iam.New(session.New(config))

	policies := make(map[string]string)
	var marker *string
	for {
		resp, err := iam.ListPolicies(&_iam.ListPoliciesInput{
			MaxItems: types.Int64P(1000),
			Marker:   marker,
		})
		if err != nil {
			break
		}
		for _, p := range resp.Policies {
			policies[*p.PolicyName] = *p.Arn
		}
		if !_aws.BoolValue(resp.IsTruncated) {
			break
		}
		marker = resp.Marker
	}

	required := []string{
		"IAMFullAccess",
		"AmazonEC2FullAccess",
		"AmazonEC2ContainerRegistryFullAccess",
		"AmazonS3FullAccess",
		"AmazonRoute53FullAccess",
	}
	missing := make([]string, 0)
	for _, name := range required {
		if _, found = policies[name]; !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return true, "Credential missing required authorization: " + strings.Join(missing, ", ")
	} else {
		return false, ""
	}
}

// Returns true if unauthorized
func IsGceUnauthorized(project string, data map[string]string) (bool, string) {
	if project == "" {
		project = data[credential.GCEProjectID]
	}

	cred, err := json.Marshal(data)
	if err != nil {
		return true, "Failed to parse credential"
	}
	conf, err := google.JWTConfigFromJSON(cred,
		compute.ComputeScope,
		compute.DevstorageReadWriteScope,
		rupdate.ReplicapoolScope)
	if err != nil {
		return true, err.Error()
	}
	client, err := compute.New(conf.Client(oauth2.NoContext))
	if err != nil {
		return true, err.Error()
	}
	_, err = client.InstanceGroups.List(project, "us-central1-b").Do()
	if err != nil {
		return true, "Credential missing required authorization"
	}
	return false, ""
}

// Returns true if unauthorized
func IsDigitalOceanUnauthorized(data map[string]string) (bool, string) {
	var token string
	var found bool
	if token, found = data[credential.DigitalOceanToken]; !found {
		return true, "Credential missing " + credential.DigitalOceanToken
	}

	client := godo.NewClient(oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})))

	name := "check-write-access:" + strconv.FormatInt(time.Now().Unix(), 10)
	_, _, err := client.Tags.Create(context.TODO(), &godo.TagCreateRequest{
		Name: name,
	})
	if err != nil {
		return true, "Credential missing WRITE scope"
	}
	client.Tags.Delete(context.TODO(), name)
	return false, ""
}

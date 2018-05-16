package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

func TestRegion(t *testing.T) {
	g, err := NewClient("us-east-1", "", "", "1.1.1")
	if err != nil {
		t.Error(err)
		return
	}
	g.Session, err = session.NewSession(&aws.Config{
		Region:      string_ptr("us-east-1"),
		Credentials: credentials.NewStaticCredentials("", "", ""),
	})
	if err != nil {
		t.Error(err)
		return
	}
	_, err = g.GetRegions()
	if err != nil {
		t.Error(err)
		return
	}
}

func TestInstance(t *testing.T) {
	g, err := NewClient("us-east-1", "", "", "1.1.1")
	if err != nil {
		t.Error(err)
		return
	}
	_, err = g.GetInstanceTypes()
	if err != nil {
		t.Error(err)
		return
	}
}

func string_ptr(in string) *string {
	var out *string
	out = &in
	return out
}

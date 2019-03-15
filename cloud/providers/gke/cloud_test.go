package gke

import (
	"context"
	"fmt"
	"testing" //. "github.com/pharmer/pharmer/cloud"

	"golang.org/x/oauth2/google"
	container "google.golang.org/api/container/v1"
	//core "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCL(t *testing.T) {
	data := ``
	project := "k8s-qa"
	conf, err := google.JWTConfigFromJSON([]byte(data),
		container.CloudPlatformScope)
	fmt.Println(err)
	client := conf.Client(context.Background())
	containerService, _ := container.New(client)
	resp, err := containerService.Projects.Zones.Clusters.Get(project, "us-central1-f", "gk5").Context(context.Background()).Do()
	fmt.Println(resp, err)
}

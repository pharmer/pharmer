package gce

import (
	"context"
	"fmt"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	rupdate "google.golang.org/api/replicapoolupdater/v1beta1"
	gcs "google.golang.org/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const containerOsImage = "appscode-containeros"

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster

	computeService *compute.Service
	storageService *gcs.Service
	updateService  *rupdate.Service
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	// TODO: FixIt cluster.Spec.Cloud.Project
	namer := namer{cluster: cluster}
	cluster.Spec.Cloud.GCE = &api.GoogleSpec{
		CloudConfig: &api.GCECloudConfig{
			// TokenURL           :
			// TokenBody          :
			ProjectID:          cluster.Spec.Cloud.Project,
			NetworkName:        "default",
			NodeTags:           []string{namer.NodePrefix()},
			NodeInstancePrefix: namer.NodePrefix(),
			Multizone:          bool(cluster.Spec.Multizone),
		},
	}

	cluster.Spec.Cloud.Project = typed.ProjectID()
	cluster.Spec.Cloud.CloudConfigPath = "/etc/gce.conf"

	conf, err := google.JWTConfigFromJSON([]byte(typed.ServiceAccount()),
		compute.ComputeScope,
		compute.DevstorageReadWriteScope,
		rupdate.ReplicapoolScope)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	client := conf.Client(context.Background())
	computeService, err := compute.New(client)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	storageService, err := gcs.New(client)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	updateService, err := rupdate.New(client)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	conn := cloudConnector{
		ctx:            ctx,
		cluster:        cluster,
		computeService: computeService,
		storageService: storageService,
		updateService:  updateService,
	}
	if ok, msg := conn.IsUnauthorized(typed.ProjectID()); !ok {
		return nil, fmt.Errorf("Credential %s does not have necessary authorization. Reason: %s.", cluster.Spec.CredentialName, msg)
	}
	return &conn, nil
}

// Returns true if unauthorized
func (conn *cloudConnector) IsUnauthorized(project string) (bool, string) {
	_, err := conn.computeService.InstanceGroups.List(project, "us-central1-b").Do()
	if err != nil {
		return false, "Credential missing required authorization"
	}
	return true, ""
}

func (conn *cloudConnector) deleteInstance(name string) error {
	Logger(conn.ctx).Info("Deleting instance...")
	r, err := conn.computeService.Instances.Delete(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForZoneOperation(r.Name)
	return nil
}

func (conn *cloudConnector) waitForGlobalOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.GlobalOperations.Get(conn.cluster.Spec.Cloud.Project, operation).Do()
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) waitForRegionOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.RegionOperations.Get(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Region, operation).Do()
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, operation)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) waitForZoneOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.ZoneOperations.Get(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, operation).Do()
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, operation)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

package gce

import (
	"context"
	"fmt"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	rupdate "google.golang.org/api/replicapoolupdater/v1beta1"
	gcs "google.golang.org/api/storage/v1"
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
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	// TODO: FixIt cluster.Spec.Project

	conf, err := google.JWTConfigFromJSON([]byte(typed.ServiceAccount()),
		compute.ComputeScope,
		compute.DevstorageReadWriteScope,
		rupdate.ReplicapoolScope)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	client := conf.Client(oauth2.NoContext)
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
		return true, "Credential missing required authorization"
	}
	return false, ""
}

func (conn *cloudConnector) getInstanceImage() (string, error) {
	_, err := conn.computeService.Images.Get(conn.cluster.Spec.Project, containerOsImage).Do()

	if err != nil {
		if !conn.checkDiskExists(containerOsImage) {
			err = conn.createImageDisk(containerOsImage)
			if err != nil {
				return "", errors.FromErr(err).WithContext(conn.ctx).Err()
			}
		}

		cloud.Logger(conn.ctx).Infof("Creating %v image on %v project", containerOsImage, conn.cluster.Spec.Project)
		r, err := conn.computeService.Images.Insert(conn.cluster.Spec.Project, &compute.Image{
			Name:       containerOsImage,
			SourceDisk: fmt.Sprintf("projects/%v/zones/%v/disks/%v", conn.cluster.Spec.Project, conn.cluster.Spec.Zone, containerOsImage),
		}).Do()
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		err = conn.waitForGlobalOperation(r.Name)
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		cloud.Logger(conn.ctx).Infof("Image %v created", containerOsImage)
		for {
			fmt.Print("Pending image creation: ")
			if r, err := conn.computeService.Images.Get(conn.cluster.Spec.Project, containerOsImage).Do(); err != nil || r.Status != "READY" {
				fmt.Print("~")
				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}

		_, err = conn.computeService.Disks.Delete(conn.cluster.Spec.Project, conn.cluster.Spec.Zone, containerOsImage).Do()
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
	}
	cloud.Logger(conn.ctx).Infof("Image %v found on project %v", containerOsImage, conn.cluster.Spec.Project)
	return containerOsImage, nil
}

func (conn *cloudConnector) checkDiskExists(name string) bool {
	_, err := conn.computeService.Disks.Get(conn.cluster.Spec.Project, conn.cluster.Spec.Zone, name).Do()
	if err != nil {
		return false
	}
	return true
}

func (conn *cloudConnector) getSrcImage() (string, error) {
	resp, err := conn.computeService.Images.List("debian-cloud").Do()
	if err != nil {
		return "", err
	}
	for _, image := range resp.Items {
		if image.Family == "debian-8" && image.Deprecated == nil {
			return image.Name, nil
		}
	}
	return "", errors.New("Failed to detect debian-8 base image.").WithContext(conn.ctx).Err()
}

func (conn *cloudConnector) createImageDisk(name string) error {
	machineType := fmt.Sprintf("projects/%v/zones/%v/machineTypes/n1-standard-1", conn.cluster.Spec.Project, conn.cluster.Spec.Zone)
	img, err := conn.getSrcImage()
	if err != nil || img == "" {
		return errors.FromErr(err).WithMessage("No debian image found").WithContext(conn.ctx).Err()
	}

	srcImage := "projects/debian-cloud/global/images/" + img
	startupScript := `#!/bin/bash
sed -i 's/GRUB_CMDLINE_LINUX="console=ttyS0,38400n8 elevator=noop"/GRUB_CMDLINE_LINUX="console=ttyS0,38400n8 elevator=noop cgroup_enable=memory swapaccount=1"/' /etc/default/grub
update-grub`
	tempInstance := &compute.Instance{
		Name:        name,
		MachineType: machineType,
		Disks: []*compute.AttachedDisk{
			{
				DeviceName: name,
				Boot:       true,
				AutoDelete: false,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: srcImage,
				},
				Mode: "READ_WRITE",
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Network: fmt.Sprintf("projects/%v/global/networks/%v", conn.cluster.Spec.Project, "default"),
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: "default",
				Scopes: []string{
					compute.DevstorageReadWriteScope,
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "startup-script",
					Value: &startupScript,
				},
			},
		},
	}
	cloud.Logger(conn.ctx).Info("Creating instance for disk...")
	r, err := conn.computeService.Instances.Insert(conn.cluster.Spec.Project, conn.cluster.Spec.Zone, tempInstance).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForZoneOperation(r.Name)
	time.Sleep(1 * time.Minute)
	cloud.Logger(conn.ctx).Info("Restarting instance ...")
	r, err = conn.computeService.Instances.Reset(conn.cluster.Spec.Project, conn.cluster.Spec.Zone, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForZoneOperation(r.Name)
	time.Sleep(1 * time.Minute)
	err = conn.deleteInstance(name)
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	return nil
}

func (conn *cloudConnector) deleteInstance(name string) error {
	cloud.Logger(conn.ctx).Info("Deleting instance...")
	r, err := conn.computeService.Instances.Delete(conn.cluster.Spec.Project, conn.cluster.Spec.Zone, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForZoneOperation(r.Name)
	return nil
}

func (conn *cloudConnector) waitForGlobalOperation(operation string) error {
	attempt := 0
	for {
		cloud.Logger(conn.ctx).Infof("Attempt %v: waiting for operation %v to complete", attempt, operation)
		r1, err := conn.computeService.GlobalOperations.Get(conn.cluster.Spec.Project, operation).Do()
		cloud.Logger(conn.ctx).Debug("Retrieved operation", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if r1.Status == "DONE" {
			return nil
		}
		cloud.Logger(conn.ctx).Infof("Attempt %v: operation %v is %v, waiting...", attempt, operation, r1.Status)
		attempt++
		time.Sleep(5 * time.Second)
		if attempt > 120 {
			break // 10 mins
		}
	}
	return errors.Newf("Global operation: %v failed to complete in 10 mins", operation).WithContext(conn.ctx).Err()
}

func (conn *cloudConnector) waitForRegionOperation(operation string) error {
	attempt := 0
	for {
		cloud.Logger(conn.ctx).Infof("Attempt %v: waiting for operation %v to complete", attempt, operation)
		r1, err := conn.computeService.RegionOperations.Get(conn.cluster.Spec.Project, conn.cluster.Spec.Region, operation).Do()
		cloud.Logger(conn.ctx).Debug("Retrieved operation", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}

		if r1.Status == "DONE" {
			return nil
		}
		cloud.Logger(conn.ctx).Infof("Attempt %v: operation %v is %v, waiting...", attempt, operation, r1.Status)
		attempt++
		time.Sleep(5 * time.Second)
		if attempt > 120 {
			break // 10 mins
		}
	}
	return errors.Newf("Region operation: %v failed to complete in 10 mins", operation).WithContext(conn.ctx).Err()
}

func (conn *cloudConnector) waitForZoneOperation(operation string) error {
	attempt := 0
	for {
		cloud.Logger(conn.ctx).Infof("Attempt %v: waiting for operation %v to complete", attempt, operation)
		r1, err := conn.computeService.ZoneOperations.Get(conn.cluster.Spec.Project, conn.cluster.Spec.Zone, operation).Do()
		cloud.Logger(conn.ctx).Debug("Retrieved operation", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}

		if r1.Status == "DONE" {
			return nil //TODO check return value
		}
		cloud.Logger(conn.ctx).Infof("Attempt %v: operation %v is %v, waiting...", attempt, operation, r1.Status)
		attempt++
		time.Sleep(5 * time.Second)
		if attempt > 120 {
			break // 10 mins
		}
	}
	return errors.Newf("Zone operation: %v failed to complete in 10 mins", operation).WithContext(conn.ctx).Err()
}

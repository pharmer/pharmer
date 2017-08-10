package gce

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	rupdate "google.golang.org/api/replicapoolupdater/v1beta1"
	gcs "google.golang.org/api/storage/v1"
)

const containerOsImage = "appscode-containeros"

type cloudConnector struct {
	ctx *contexts.ClusterContext

	computeService *compute.Service
	storageService *gcs.Service
	updateService  *rupdate.Service
}

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	var err error
	credGCP, err := json.Marshal(ctx.CloudCredential)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	conf, err := google.JWTConfigFromJSON(credGCP,
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

	return &cloudConnector{
		ctx:            ctx,
		computeService: computeService,
		storageService: storageService,
		updateService:  updateService,
	}, nil
}

func (conn *cloudConnector) getInstanceImage() (string, error) {
	_, err := conn.computeService.Images.Get(conn.ctx.Project, containerOsImage).Do()

	if err != nil {
		if !conn.checkDiskExists(containerOsImage) {
			err = conn.createImageDisk(containerOsImage)
			if err != nil {
				return "", errors.FromErr(err).WithContext(conn.ctx).Err()
			}
		}

		conn.ctx.Logger().Infof("Creating %v image on %v project", containerOsImage, conn.ctx.Project))
		r, err := conn.computeService.Images.Insert(conn.ctx.Project, &compute.Image{
			Name:       containerOsImage,
			SourceDisk: fmt.Sprintf("projects/%v/zones/%v/disks/%v", conn.ctx.Project, conn.ctx.Zone, containerOsImage),
		}).Do()
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		err = conn.waitForGlobalOperation(r.Name)
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		conn.ctx.Logger().Infof("Image %v created", containerOsImage))
		for {
			fmt.Print("Pending image creation: ")
			if r, err := conn.computeService.Images.Get(conn.ctx.Project, containerOsImage).Do(); err != nil || r.Status != "READY" {
				fmt.Print("~")
				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}

		_, err = conn.computeService.Disks.Delete(conn.ctx.Project, conn.ctx.Zone, containerOsImage).Do()
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
	}
	conn.ctx.Logger().Infof("Image %v found on project %v", containerOsImage, conn.ctx.Project))
	return containerOsImage, nil
}

func (conn *cloudConnector) checkDiskExists(name string) bool {
	_, err := conn.computeService.Disks.Get(conn.ctx.Project, conn.ctx.Zone, name).Do()
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
	machineType := fmt.Sprintf("projects/%v/zones/%v/machineTypes/n1-standard-1", conn.ctx.Project, conn.ctx.Zone)
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
				Network: fmt.Sprintf("projects/%v/global/networks/%v", conn.ctx.Project, "default"),
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
	conn.ctx.Logger().Info("Creating instance for disk...")
	r, err := conn.computeService.Instances.Insert(conn.ctx.Project, conn.ctx.Zone, tempInstance).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForZoneOperation(r.Name)
	time.Sleep(1 * time.Minute)
	conn.ctx.Logger().Info("Restarting instance ...")
	r, err = conn.computeService.Instances.Reset(conn.ctx.Project, conn.ctx.Zone, name).Do()
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
	conn.ctx.Logger().Info("Deleting instance...")
	r, err := conn.computeService.Instances.Delete(conn.ctx.Project, conn.ctx.Zone, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForZoneOperation(r.Name)
	return nil
}

func (conn *cloudConnector) waitForGlobalOperation(operation string) error {
	attempt := 0
	for {
		conn.ctx.Logger().Infof("Attempt %v: waiting for operation %v to complete", attempt, operation)
		r1, err := conn.computeService.GlobalOperations.Get(conn.ctx.Project, operation).Do()
		conn.ctx.Logger().Debug("Retrieved operation", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if r1.Status == "DONE" {
			return nil
		}
		conn.ctx.Logger().Infof("Attempt %v: operation %v is %v, waiting...", attempt, operation, r1.Status))
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
		conn.ctx.Logger().Infof("Attempt %v: waiting for operation %v to complete", attempt, operation)
		r1, err := conn.computeService.RegionOperations.Get(conn.ctx.Project, conn.ctx.Region, operation).Do()
		conn.ctx.Logger().Debug("Retrieved operation", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}

		if r1.Status == "DONE" {
			return nil
		}
		conn.ctx.Logger().Infof("Attempt %v: operation %v is %v, waiting...", attempt, operation, r1.Status))
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
		conn.ctx.Logger().Infof("Attempt %v: waiting for operation %v to complete", attempt, operation)
		r1, err := conn.computeService.ZoneOperations.Get(conn.ctx.Project, conn.ctx.Zone, operation).Do()
		conn.ctx.Logger().Debug("Retrieved operation", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}

		if r1.Status == "DONE" {
			return nil //TODO check return value
		}
		conn.ctx.Logger().Infof("Attempt %v: operation %v is %v, waiting...", attempt, operation, r1.Status))
		attempt++
		time.Sleep(5 * time.Second)
		if attempt > 120 {
			break // 10 mins
		}
	}
	return errors.Newf("Zone operation: %v failed to complete in 10 mins", operation).WithContext(conn.ctx).Err()
}

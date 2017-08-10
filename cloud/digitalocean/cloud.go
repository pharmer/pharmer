package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/credential"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

const containerOsImage = "appscode-containeros"

type cloudConnector struct {
	ctx    *contexts.ClusterContext
	client *godo.Client
}

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	token, ok := ctx.CloudCredential[credential.DigitalOceanToken]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credential.DigitalOceanToken)
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	}))
	return &cloudConnector{
		ctx:    ctx,
		client: godo.NewClient(oauthClient),
	}, nil
}

func (conn *cloudConnector) getInstanceImage() (string, error) {
	imgPage := 0
	const imgPageSize = 20
	for {
		imgs, _, err := conn.client.Images.ListUser(context.TODO(), &godo.ListOptions{
			Page:    imgPage,
			PerPage: imgPageSize,
		})
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		conn.ctx.Logger.Debugln("List user images")
		for _, img := range imgs {
			conn.ctx.Logger.Debugln(img.ID, "__", img.Name, "---", img.Distribution, "---", img.Type)
			if img.Name == containerOsImage && img.Distribution == "Debian" {
				found := false
				for _, region := range img.Regions {
					if region == conn.ctx.Region {
						found = true
						conn.ctx.Logger.Debugf("Image already exists in region %v.", conn.ctx.Region)
						return strconv.Itoa(img.ID), nil
					}
				}
				if !found {
					_, _, err := conn.client.ImageActions.Transfer(context.TODO(), img.ID, &godo.ActionRequest{
						"type":   "transfer",
						"region": conn.ctx.Region,
					})
					if err != nil {
						return "", errors.FromErr(err).WithContext(conn.ctx).Err()
					}

					conn.ctx.Logger.Infof("Started image transfer to region %v.", conn.ctx.Region)
					// wait for the transfer to complete
					conn.waitForTransfer(img.ID)
					return strconv.Itoa(img.ID), nil
				}
			}
		}
		imgPage++
		if len(imgs) < imgPageSize {
			break
		}
	}

	conn.ctx.Logger.Info("Creating droplet to build custom image")
	droplet, _, err := conn.client.Droplets.Create(context.TODO(), &godo.DropletCreateRequest{
		Name:              rand.WithUniqSuffix("kubernetes"),
		Region:            conn.ctx.Region,
		Size:              "512mb",
		Image:             godo.DropletCreateImage{Slug: "debian-8-x64"},
		PrivateNetworking: false,
		IPv6:              false,
		// http://serverfault.com/a/783795/167143
		UserData: `#!/bin/bash
sed -i 's/^GRUB_CMDLINE_LINUX="/GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1 /' /etc/default/grub
sed -i 's/^GRUB_CMDLINE_LINUX="/GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1 /' /etc/default/grub.d/50-cloudimg-settings.cfg
update-grub`,
	})
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.ctx.Logger.Info("Wait for custom image instance to become active")
	conn.waitForInstance(droplet.ID, "active")
	time.Sleep(30 * time.Second)

	conn.ctx.Logger.Info("Power off custom image instance")
	_, _, err = conn.client.DropletActions.PowerOff(context.TODO(), droplet.ID)
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForInstance(droplet.ID, "off")

	conn.ctx.Logger.Info("Start taking custom image snapshot")
	_, _, err = conn.client.DropletActions.Snapshot(context.TODO(), droplet.ID, containerOsImage)
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	conn.ctx.Logger.Info("Wait for custom image snapshot to be completed")
	for {
		action, err := conn.findSnapshotAction(droplet.ID)
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if action.Status == "completed" {
			break
		}
		conn.ctx.Logger.Debugln(".")
		time.Sleep(5 * time.Second)
	}

	var k8sImage *godo.Image
	snaps, _, err := conn.client.Droplets.Snapshots(context.TODO(), droplet.ID, &godo.ListOptions{
		Page:    0,
		PerPage: 1, // there can be only one snapshot for the new custom image droplet
	})
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	if len(snaps) == 1 {
		if snaps[0].Name == containerOsImage {
			k8sImage = &snaps[0]
		}
	}
	conn.ctx.Logger.Debugln("New K8s base image id", k8sImage.ID, ", name: ", k8sImage.Name)

	_, err = conn.client.Droplets.Delete(context.TODO(), droplet.ID)
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.ctx.Logger.Info("Delete custom image instance")
	return strconv.Itoa(k8sImage.ID), nil
}

func (conn *cloudConnector) waitForInstance(id int, status string) error {
	attempt := 0
	for true {
		conn.ctx.Logger.Infof("Checking status of instance %v", id)
		droplet, _, err := conn.client.Droplets.Get(context.TODO(), id)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		conn.ctx.Logger.Debugf("Instance status %v, %v", droplet, err)
		if strings.ToLower(droplet.Status) == status {
			break
		}
		conn.ctx.Logger.Infof("Instance %v (%v) is %v, waiting...", droplet.Name, droplet.ID, droplet.Status)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil
}

func (conn *cloudConnector) findSnapshotAction(id int) (*godo.Action, error) {
	actions, _, err := conn.client.Droplets.Actions(context.TODO(), id, &godo.ListOptions{
		Page:    0,
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}
	for _, a := range actions {
		if a.Type == "snapshot" {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("snapshot action not found for droplet %v", id)
}

func (conn *cloudConnector) waitForTransfer(id int) error {
	conn.ctx.Logger.Infof("Wait for the transfer to complete")
	attempt := 0
	for {
		img, _, err := conn.client.Images.GetByID(context.TODO(), id)
		if err != nil {
			return err
		}
		for _, r := range img.Regions {
			if r == conn.ctx.Region {
				return nil
			}
		}
		conn.ctx.Logger.Debug(".")
		attempt++
		time.Sleep(10 * time.Second)
		if attempt > 60 {
			break
		}
	}
	return errors.New("Failed to transfer container os image.").WithContext(conn.ctx).Err()
}

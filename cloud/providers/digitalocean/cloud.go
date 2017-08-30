package digitalocean

import (
	"context"
	gtx "context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

const containerOsImage = "appscode-containeros"

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *godo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: typed.Token(),
	}))
	conn := cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  godo.NewClient(oauthClient),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, fmt.Errorf("Credential %s does not have necessary autheorization. Reason: %s.", cluster.Spec.CredentialName, msg)
	}
	return &conn, nil
}

// Returns true if unauthorized
func (conn *cloudConnector) IsUnauthorized() (bool, string) {
	name := "check-write-access:" + strconv.FormatInt(time.Now().Unix(), 10)
	_, _, err := conn.client.Tags.Create(gtx.TODO(), &godo.TagCreateRequest{
		Name: name,
	})
	if err != nil {
		return true, "Credential missing WRITE scope"
	}
	conn.client.Tags.Delete(gtx.TODO(), name)
	return false, ""
}

func (conn *cloudConnector) getInstanceImage() (string, error) {
	imgPage := 0
	const imgPageSize = 20
	for {
		imgs, _, err := conn.client.Images.ListUser(gtx.TODO(), &godo.ListOptions{
			Page:    imgPage,
			PerPage: imgPageSize,
		})
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		cloud.Logger(conn.ctx).Debugln("List user images")
		for _, img := range imgs {
			cloud.Logger(conn.ctx).Debugln(img.ID, "__", img.Name, "---", img.Distribution, "---", img.Type)
			if img.Name == containerOsImage && img.Distribution == "Debian" {
				found := false
				for _, region := range img.Regions {
					if region == conn.cluster.Spec.Region {
						found = true
						cloud.Logger(conn.ctx).Debugf("Image already exists in region %v.", conn.cluster.Spec.Region)
						return strconv.Itoa(img.ID), nil
					}
				}
				if !found {
					_, _, err := conn.client.ImageActions.Transfer(gtx.TODO(), img.ID, &godo.ActionRequest{
						"type":   "transfer",
						"region": conn.cluster.Spec.Region,
					})
					if err != nil {
						return "", errors.FromErr(err).WithContext(conn.ctx).Err()
					}

					cloud.Logger(conn.ctx).Infof("Started image transfer to region %v.", conn.cluster.Spec.Region)
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

	cloud.Logger(conn.ctx).Info("Creating droplet to build custom image")
	droplet, _, err := conn.client.Droplets.Create(gtx.TODO(), &godo.DropletCreateRequest{
		Name:              rand.WithUniqSuffix("kubernetes"),
		Region:            conn.cluster.Spec.Region,
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
	cloud.Logger(conn.ctx).Info("Wait for custom image instance to become active")
	conn.waitForInstance(droplet.ID, "active")
	time.Sleep(30 * time.Second)

	cloud.Logger(conn.ctx).Info("Power off custom image instance")
	_, _, err = conn.client.DropletActions.PowerOff(gtx.TODO(), droplet.ID)
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.waitForInstance(droplet.ID, "off")

	cloud.Logger(conn.ctx).Info("Start taking custom image snapshot")
	_, _, err = conn.client.DropletActions.Snapshot(gtx.TODO(), droplet.ID, containerOsImage)
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	cloud.Logger(conn.ctx).Info("Wait for custom image snapshot to be completed")
	for {
		action, err := conn.findSnapshotAction(droplet.ID)
		if err != nil {
			return "", errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if action.Status == "completed" {
			break
		}
		cloud.Logger(conn.ctx).Debugln(".")
		time.Sleep(5 * time.Second)
	}

	var k8sImage *godo.Image
	snaps, _, err := conn.client.Droplets.Snapshots(gtx.TODO(), droplet.ID, &godo.ListOptions{
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
	cloud.Logger(conn.ctx).Debugln("New K8s base image id", k8sImage.ID, ", name: ", k8sImage.Name)

	_, err = conn.client.Droplets.Delete(gtx.TODO(), droplet.ID)
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	cloud.Logger(conn.ctx).Info("Delete custom image instance")
	return strconv.Itoa(k8sImage.ID), nil
}

func (conn *cloudConnector) waitForInstance(id int, status string) error {
	attempt := 0
	for true {
		cloud.Logger(conn.ctx).Infof("Checking status of instance %v", id)
		droplet, _, err := conn.client.Droplets.Get(gtx.TODO(), id)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		cloud.Logger(conn.ctx).Debugf("Instance status %v, %v", droplet, err)
		if strings.ToLower(droplet.Status) == status {
			break
		}
		cloud.Logger(conn.ctx).Infof("Instance %v (%v) is %v, waiting...", droplet.Name, droplet.ID, droplet.Status)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil
}

func (conn *cloudConnector) findSnapshotAction(id int) (*godo.Action, error) {
	actions, _, err := conn.client.Droplets.Actions(gtx.TODO(), id, &godo.ListOptions{
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
	cloud.Logger(conn.ctx).Infof("Wait for the transfer to complete")
	attempt := 0
	for {
		img, _, err := conn.client.Images.GetByID(gtx.TODO(), id)
		if err != nil {
			return err
		}
		for _, r := range img.Regions {
			if r == conn.cluster.Spec.Region {
				return nil
			}
		}
		cloud.Logger(conn.ctx).Debug(".")
		attempt++
		time.Sleep(10 * time.Second)
		if attempt > 60 {
			break
		}
	}
	return errors.New("Failed to transfer container os image.").WithContext(conn.ctx).Err()
}

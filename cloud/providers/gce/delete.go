package gce

import (
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	. "github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) deleteMaster() error {
	r2, err := cm.conn.computeService.Instances.Delete(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, cm.cluster.Spec.KubernetesMasterName).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	operation := r2.Name
	cm.conn.waitForZoneOperation(operation)
	Logger(cm.ctx).Infof("Master instance %v deleted", cm.cluster.Spec.KubernetesMasterName)
	return nil

}

//delete instance group
func (cm *ClusterManager) deleteNodeGroup(instanceGroup string) error {
	r1, err := cm.conn.computeService.InstanceGroupManagers.Delete(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, instanceGroup).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	operation := r1.Name
	cm.conn.waitForZoneOperation(operation)
	Logger(cm.ctx).Infof("Instance group %v deleted", instanceGroup)
	return nil
}

//delete template
func (cm *ClusterManager) deleteInstanceTemplate(template string) error {
	_, err := cm.conn.computeService.InstanceTemplates.Delete(cm.cluster.Spec.Cloud.Project, template).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Instance templete %v deleted", template)
	//cluster.Spec.waitForGlobalOperation(r.Name)
	return nil
}

//delete autoscaler
func (cm *ClusterManager) deleteAutoscaler(instanceGroup string) error {
	Logger(cm.ctx).Infof("Removing autoscaller %v", instanceGroup)

	r, err := cm.conn.computeService.Autoscalers.Delete(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, instanceGroup).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForZoneOperation(r.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Autoscaller %v is deleted", instanceGroup)
	return nil
}

//delete disk
func (cm *ClusterManager) deleteDisk() error {
	masterDisk := cm.namer.MasterPDName()
	r6, err := cm.conn.computeService.Disks.Delete(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, masterDisk).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Debugf("Master Disk response %v", r6)
	time.Sleep(5 * time.Second)
	r7, err := cm.conn.computeService.Disks.List(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for i := range r7.Items {
		s := strings.Split(r7.Items[i].Name, "-")
		if s[0] == cm.cluster.Name {

			r, err := cm.conn.computeService.Disks.Delete(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, r7.Items[i].Name).Do()
			if err != nil {
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			Logger(cm.ctx).Infof("Disk %v deleted, response %v", r7.Items[i].Name, r.Status)
			time.Sleep(5 * time.Second)
		}

	}
	return nil
}

//delete firewalls
func (cm *ClusterManager) deleteFirewalls() error {
	name := cm.cluster.Name + "-node-all"
	r1, err := cm.conn.computeService.Firewalls.Delete(cm.cluster.Spec.Cloud.Project, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Firewalls %v deleted, response %v", name, r1.Status)
	//cluster.Spec.waitForGlobalOperation(name)
	time.Sleep(5 * time.Second)
	ruleHTTPS := cm.cluster.Spec.KubernetesMasterName + "-https"
	r2, err := cm.conn.computeService.Firewalls.Delete(cm.cluster.Spec.Cloud.Project, ruleHTTPS).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Firewalls %v deleted, response %v", ruleHTTPS, r2.Status)
	//cluster.Spec.waitForGlobalOperation(ruleHTTPS)
	time.Sleep(5 * time.Second)
	return nil
}

// delete reserve ip
func (cm *ClusterManager) releaseReservedIP() error {
	name := cm.namer.ReserveIPName()
	r1, err := cm.conn.computeService.Addresses.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Region, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Releasing reserved master ip %v", r1.Address)
	r2, err := cm.conn.computeService.Addresses.Delete(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Region, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForRegionOperation(r2.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Master ip %v released", r1.Address)
	return nil
}

func (cm *ClusterManager) deleteRoutes() error {
	r1, err := cm.conn.computeService.Routes.List(cm.cluster.Spec.Cloud.Project).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for i := range r1.Items {
		routeName := r1.Items[i].Name
		if strings.HasPrefix(routeName, cm.cluster.Name) {
			fmt.Println(routeName)
			r2, err := cm.conn.computeService.Routes.Delete(cm.cluster.Spec.Cloud.Project, routeName).Do()
			if err != nil {
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			Logger(cm.ctx).Infof("Route %v deleted, response %v", routeName, r2.Status)
		}
	}
	return nil
}

func (cm *ClusterManager) deleteSSHKey() (err error) {
	//if cm.cluster.Spec.SSHKeyPHID != "" {
	//	//updates := &storage.SSHKey{IsDeleted: 1}
	//	//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
	//	//_, err = Store(cm.ctx).Engine.Update(updates, cond)
	//	//cm.ctx.Notifier.StoreAndNotify(api.JobPhaseRunning, fmt.Sprintf("SSH key for cluster %v deleted", cm.ctx.MasterDiskId))
	//}
	return
}

package gce

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *clusterManager) delete(req *proto.ClusterDeleteRequest) error {
	defer cm.cluster.Delete()

	if cm.cluster.Status.Phase == api.ClusterPhasePending {
		cm.cluster.Status.Phase = api.ClusterPhaseFailing
	} else if cm.cluster.Status.Phase == api.ClusterPhaseReady {
		cm.cluster.Status.Phase = api.ClusterPhaseDeleting
	}
	// cm.ctx.Store().UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}
	cm.namer = namer{cluster: cm.cluster}

	var errs []string
	if cm.cluster.Status.Reason != "" {
		errs = append(errs, cm.cluster.Status.Reason)
	}

	if l, err := cm.listInstanceGroups(); err == nil {
		for _, g := range l {
			instanceGroup := g.groupName
			template := cm.namer.InstanceTemplateName(g.sku)

			if err = cm.deleteInstanceGroup(instanceGroup); err != nil {
				errs = append(errs, err.Error())
			}

			if err = cm.deleteAutoscaler(instanceGroup); err != nil {
				errs = append(errs, err.Error())
			}

			if err = cm.deleteInstanceTemplate(template); err != nil {
				errs = append(errs, err.Error())
			}
		}
	} else {
		errs = append(errs, err.Error())
	}
	if err := cm.deleteMaster(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := cm.deleteFirewalls(); err != nil {
		errs = append(errs, err.Error())
	}
	if req.ReleaseReservedIp {
		if err := cm.releaseReservedIP(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if err := cm.deleteDisk(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := cm.deleteRoutes(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := cm.deleteBucket(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := cloud.DeleteARecords(cm.ctx, cm.cluster); err != nil {
		errs = append(errs, err.Error())
	}

	// Delete SSH key from DB
	if err := cm.deleteSSHKey(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status.Phase == api.ClusterPhaseDeleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	cm.ctx.Logger().Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

type groupInfo struct {
	groupName string
	sku       string
}

func (cm *clusterManager) listInstanceGroups() ([]*groupInfo, error) {
	groups := make([]*groupInfo, 0)

	r1, err := cm.conn.computeService.InstanceGroups.List(cm.cluster.Spec.Project, cm.cluster.Spec.Zone).Do()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for _, g := range r1.Items {
		name := g.Name
		if strings.HasPrefix(name, cm.cluster.Name) {
			groups = append(groups, &groupInfo{
				groupName: name,
				sku:       strings.TrimSuffix(strings.TrimPrefix(name, cm.cluster.Name+"-"), "-v"+strconv.FormatInt(cm.cluster.Spec.ResourceVersion, 10)),
			})
		}

	}
	if len(groups) == 0 {
		cm.ctx.Logger().Info("Enter correct cluster name")
		//os.Exit(1)
	}
	cm.ctx.Logger().Debugf("Retrieved InstanceGroups result %v", groups)
	return groups, nil
}

func (cm *clusterManager) deleteMaster() error {
	r2, err := cm.conn.computeService.Instances.Delete(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, cm.cluster.Spec.KubernetesMasterName).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	operation := r2.Name
	cm.conn.waitForZoneOperation(operation)
	cm.ctx.Logger().Infof("Master instance %v deleted", cm.cluster.Spec.KubernetesMasterName)
	return nil

}

//delete instance group
func (cm *clusterManager) deleteInstanceGroup(instanceGroup string) error {
	r1, err := cm.conn.computeService.InstanceGroupManagers.Delete(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instanceGroup).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	operation := r1.Name
	cm.conn.waitForZoneOperation(operation)
	cm.ctx.Logger().Infof("Instance group %v deleted", instanceGroup)
	return nil
}

//delete template
func (cm *clusterManager) deleteInstanceTemplate(template string) error {
	_, err := cm.conn.computeService.InstanceTemplates.Delete(cm.cluster.Spec.Project, template).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Instance templete %v deleted", template)
	//cluster.Spec.waitForGlobalOperation(r.Name)
	return nil
}

//delete autoscaler
func (cm *clusterManager) deleteAutoscaler(instanceGroup string) error {
	cm.ctx.Logger().Infof("Removing autoscaller %v", instanceGroup)

	r, err := cm.conn.computeService.Autoscalers.Delete(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instanceGroup).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForZoneOperation(r.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Autoscaller %v is deleted", instanceGroup)
	return nil
}

//delete disk
func (cm *clusterManager) deleteDisk() error {
	masterDisk := cm.namer.MasterPDName()
	r6, err := cm.conn.computeService.Disks.Delete(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, masterDisk).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Debugf("Master Disk response %v", r6)
	time.Sleep(5 * time.Second)
	r7, err := cm.conn.computeService.Disks.List(cm.cluster.Spec.Project, cm.cluster.Spec.Zone).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for i := range r7.Items {
		s := strings.Split(r7.Items[i].Name, "-")
		if s[0] == cm.cluster.Name {

			r, err := cm.conn.computeService.Disks.Delete(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, r7.Items[i].Name).Do()
			if err != nil {
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			cm.ctx.Logger().Infof("Disk %v deleted, response %v", r7.Items[i].Name, r.Status)
			time.Sleep(5 * time.Second)
		}

	}
	return nil
}

//delete firewalls
func (cm *clusterManager) deleteFirewalls() error {
	name := cm.cluster.Name + "-node-all"
	r1, err := cm.conn.computeService.Firewalls.Delete(cm.cluster.Spec.Project, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Firewalls %v deleted, response %v", name, r1.Status)
	//cluster.Spec.waitForGlobalOperation(name)
	time.Sleep(5 * time.Second)
	ruleHTTPS := cm.cluster.Spec.KubernetesMasterName + "-https"
	r2, err := cm.conn.computeService.Firewalls.Delete(cm.cluster.Spec.Project, ruleHTTPS).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Firewalls %v deleted, response %v", ruleHTTPS, r2.Status)
	//cluster.Spec.waitForGlobalOperation(ruleHTTPS)
	time.Sleep(5 * time.Second)
	return nil
}

// delete reserve ip
func (cm *clusterManager) releaseReservedIP() error {
	name := cm.namer.ReserveIPName()
	r1, err := cm.conn.computeService.Addresses.Get(cm.cluster.Spec.Project, cm.cluster.Spec.Region, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Releasing reserved master ip %v", r1.Address)
	r2, err := cm.conn.computeService.Addresses.Delete(cm.cluster.Spec.Project, cm.cluster.Spec.Region, name).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForRegionOperation(r2.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Master ip %v released", r1.Address)
	return nil
}

func (cm *clusterManager) deleteRoutes() error {
	r1, err := cm.conn.computeService.Routes.List(cm.cluster.Spec.Project).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for i := range r1.Items {
		routeName := r1.Items[i].Name
		if strings.HasPrefix(routeName, cm.cluster.Name) {
			fmt.Println(routeName)
			r2, err := cm.conn.computeService.Routes.Delete(cm.cluster.Spec.Project, routeName).Do()
			if err != nil {
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			cm.ctx.Logger().Infof("Route %v deleted, response %v", routeName, r2.Status)
		}
	}
	return nil
}

func (cm *clusterManager) deleteBucket() error {
	var timeout int64 = 30 * 60 // Give max 30 min to empty the bucket
	start := time.Now().Unix()

	for {
		objs, err := cm.conn.storageService.Objects.List(cm.cluster.Spec.BucketName).Do()
		if err == nil {
			for _, obj := range objs.Items {
				cm.conn.storageService.Objects.Delete(obj.Bucket, obj.Name).Do()
			}
			if len(objs.Items) == 0 || (time.Now().Unix() > start+timeout) {
				break
			}
		}
	}
	cm.ctx.Logger().Infof("Bucket %v deleted", cm.cluster.Spec.BucketName)
	return cm.conn.storageService.Buckets.Delete(cm.cluster.Spec.BucketName).Do()
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	if cm.cluster.Spec.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store().Engine.Update(updates, cond)
		//cm.ctx.Notifier.StoreAndNotify(api.JobPhaseRunning, fmt.Sprintf("SSH key for cluster %v deleted", cm.ctx.MasterDiskId))
	}
	return
}

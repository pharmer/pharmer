package gce

import (
	"fmt"
	"log"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	compute "google.golang.org/api/compute/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *clusterManager) setVersion(req *proto.ClusterReconfigureRequest) error {
	if !lib.UpgradeRequired(cm.ctx, req) {
		cm.ctx.Logger().Infof("Upgrade command skipped for cluster %v", cm.ctx.Name)
		return nil // TODO check error nil
	}

	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}

	cm.ctx.ContextVersion = int64(0)
	cm.namer = namer{ctx: cm.ctx}
	cm.updateContext()
	// assign new timestamp and new launch_config version
	cm.ctx.EnvTimestamp = time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	cm.ctx.KubeVersion = req.Version
	err := cm.ctx.Save()

	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	fmt.Println("Updating...")
	cm.ins, err = lib.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Load()
	if req.ApplyToMaster {
		err = cm.updateMaster()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	} else {
		err = cm.updateNodes(req.Sku)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	err = cm.ctx.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) updateMaster() error {
	fmt.Println("Updating Master...")
	err := cm.deleteMaster()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	img, err := cm.conn.getInstanceImage()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.InstanceImage = img
	cm.UploadStartupConfig()

	op, err := cm.createMasterIntance()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForZoneOperation(op)

	if err := lib.ProbeKubeAPI(cm.ctx); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance, err := cm.getInstance(cm.ctx.KubernetesMasterName)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// cm.ins.Instances = nil
	// cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	err = cm.ctx.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	fmt.Println(masterInstance)
	fmt.Println("Master Updated")
	return nil
}

func (cm *clusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")

	newInstanceTemplate := cm.namer.InstanceTemplateName(sku)
	op, err := cm.createNodeInstanceTemplate(sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForGlobalOperation(op)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	ctxV, err := lib.GetExistingContextVersion(cm.ctx, sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	oldInstanceTemplate := cm.namer.InstanceTemplateNameWithContext(sku, ctxV)
	groupName := cm.namer.InstanceGroupName(sku)
	oldinstances, err := cm.listInstances(groupName)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances := []string{}
	prefix := "https://www.googleapis.com/compute/v1/projects/" + cm.ctx.Project + "/zones/" + cm.ctx.Zone + "/instances/"
	for _, instance := range oldinstances {
		instanceName := prefix + instance.Name
		instances = append(instances, instanceName)
	}
	err = cm.rollingUpdate(instances, newInstanceTemplate, sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	currentIns, err := cm.listInstances(groupName)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = lib.AdjustDbInstance(cm.ins, currentIns, sku)
	// cluster.ctx.Instances = append(cluster.ctx.Instances, instances...)
	err = cm.ctx.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cm.deleteInstanceTemplate(oldInstanceTemplate)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	return nil
}

func (cm *clusterManager) getExistingContextVersion(sku string) (error, int64) {
	kc, err := cm.ctx.NewKubeClient()
	if err != nil {
		log.Fatal(err)
	}
	//re, _ := labels.NewRequirement(api.NodeLabelKey_SKU, selection.Equals, []string{sku})
	nodes, err := kc.Client.CoreV1().Nodes().List(metav1.ListOptions{
	//LabelSelector: labels.Selector.Add(*re).Matches(labels.Labels(api.NodeLabelKey_SKU)),
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, n := range nodes.Items {
		nl := api.FromMap(n.GetLabels())
		if nl.GetString(api.NodeLabelKey_SKU) == sku {
			return nil, nl.GetInt64(api.NodeLabelKey_ContextVersion)
		}
	}
	return errors.New("Context version not found").Err(), int64(0)
}

func (cm *clusterManager) rollingUpdate(oldInstances []string, newInstanceTemplate, sku string) error {
	groupName := cm.namer.InstanceGroupName(sku)
	newTemplate := fmt.Sprintf("projects/%v/global/instanceTemplates/%v", cm.ctx.Project, newInstanceTemplate)
	template := &compute.InstanceGroupManagersSetInstanceTemplateRequest{
		InstanceTemplate: newTemplate,
	}
	tmpR, err := cm.conn.computeService.InstanceGroupManagers.SetInstanceTemplate(cm.ctx.Project, cm.ctx.Zone, groupName, template).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn.waitForZoneOperation(tmpR.Name)
	fmt.Println("rolling update started...")

	// gcloud compute --project "tigerworks-kube" instance-groups managed recreate-instances  "gce153-n1-standard-2-v160" --zone "us-central1-f" --instances "gce153-node-n1-standard-2-whf8"
	for _, instance := range oldInstances {
		fmt.Println("updating ", instance)
		updates := &compute.InstanceGroupManagersRecreateInstancesRequest{
			Instances: []string{instance},
		}
		r, err := cm.conn.computeService.InstanceGroupManagers.RecreateInstances(cm.ctx.Project, cm.ctx.Zone, groupName, updates).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		cm.conn.waitForZoneOperation(r.Name)
		fmt.Println("Waiting for 1 minute")
		time.Sleep(1 * time.Minute)
		err = lib.WaitForReadyNodes(cm.ctx)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	return nil
}

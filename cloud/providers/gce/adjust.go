package gce

import (
	"fmt"
	"os"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/cloud"
	compute "google.golang.org/api/compute/v1"
)

type InstanceGroupManager struct {
	cm       *ClusterManager
	instance cloud.Instance
}

func (igm *InstanceGroupManager) AdjustInstanceGroup() error {
	instanceGroupName := igm.cm.namer.InstanceGroupName(igm.instance.Type.Sku)
	found := igm.cm.checkInstanceGroup(instanceGroupName)
	igm.cm.cluster.Spec.ResourceVersion = igm.instance.Type.ContextVersion
	igm.cm.cluster, _ = igm.cm.ctx.Store().Clusters().LoadCluster(igm.cm.cluster.Name)
	if !found {
		if op2, err := igm.createNodeInstanceTemplate(igm.instance.Type.Sku); err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		} else {
			if err = igm.cm.conn.waitForGlobalOperation(op2); err != nil {
				igm.cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		}
		if op3, err := igm.createInstanceGroup(igm.instance.Type.Sku, igm.instance.Stats.Count); err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		} else {
			if err = igm.cm.conn.waitForZoneOperation(op3); err != nil {
				igm.cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		}

		if op4, err := igm.createAutoscaler(igm.instance.Type.Sku, igm.instance.Stats.Count); err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		} else {
			if err = igm.cm.conn.waitForZoneOperation(op4); err != nil {
				igm.cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		}
	} else if igm.instance.Stats.Count == 0 {
		instanceTemplate := igm.cm.namer.InstanceTemplateName(igm.instance.Type.Sku)
		err := igm.deleteOnlyInstanceGroup(instanceGroupName, instanceTemplate)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else {
		err := igm.updateInstanceGroup(instanceGroupName, igm.instance.Stats.Count)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}

	return nil
}

func (igm *InstanceGroupManager) createNodeInstanceTemplate(sku string) (string, error) {
	templateName := igm.cm.namer.InstanceTemplateName(sku)

	igm.cm.ctx.Logger().Infof("Retrieving node template %v", templateName)
	if r1, err := igm.cm.conn.computeService.InstanceTemplates.Get(igm.cm.cluster.Spec.Project, templateName).Do(); err == nil {
		igm.cm.ctx.Logger().Debug("Retrieved node template", r1, err)

		if r2, err := igm.cm.conn.computeService.InstanceTemplates.Delete(igm.cm.cluster.Spec.Project, templateName).Do(); err != nil {
			igm.cm.ctx.Logger().Debug("Delete node template called", r2, err)
			igm.cm.ctx.Logger().Infoln("Failed to delete existing instance template")
			os.Exit(1)
		}
		igm.cm.ctx.Logger().Infof("Existing node template %v deleted", templateName)
	}
	//  if cluster.Spec.ctx.Preemptiblenode == "true" {
	//	  preemptible_nodes = "--preemptible --maintenance-policy TERMINATE"
	//  }

	igm.cm.UploadStartupConfig()
	startupScript := cloud.RenderKubeadmNodeStarter(igm.cm.cluster)

	image := fmt.Sprintf("projects/%v/global/images/%v", igm.cm.cluster.Spec.InstanceImageProject, igm.cm.cluster.Spec.InstanceImage)
	network := fmt.Sprintf("projects/%v/global/networks/%v", igm.cm.cluster.Spec.Project, defaultNetwork)

	tpl := &compute.InstanceTemplate{
		Name: templateName,
		Properties: &compute.InstanceProperties{
			MachineType: sku,
			Scheduling: &compute.Scheduling{
				AutomaticRestart:  false,
				OnHostMaintenance: "TERMINATE",
			},
			Disks: []*compute.AttachedDisk{
				{
					AutoDelete: true,
					Boot:       true,
					InitializeParams: &compute.AttachedDiskInitializeParams{
						DiskType:    igm.cm.cluster.Spec.NodeDiskType,
						DiskSizeGb:  igm.cm.cluster.Spec.NodeDiskSize,
						SourceImage: image,
					},
				},
			},
			Tags: &compute.Tags{
				Items: []string{igm.cm.cluster.Name + "-node"},
			},
			NetworkInterfaces: []*compute.NetworkInterface{
				{
					Network: network,
					//AccessConfigs: []*compute.AccessConfig{
					//	{
					//		Name: "External IP",
					//		Type: "ONE_TO_ONE_NAT",
					//	},
					//},
				},
			},
			ServiceAccounts: []*compute.ServiceAccount{
				{
					Scopes: []string{
						"https://www.googleapis.com/auth/compute",
						"https://www.googleapis.com/auth/devstorage.read_only",
						"https://www.googleapis.com/auth/logging.write",
					},
					Email: "default",
				},
			},
			CanIpForward: true,
			Metadata: &compute.Metadata{
				Items: []*compute.MetadataItems{
					{
						Key:   "startup-script",
						Value: &startupScript,
					},
				},
			},
		},
	}
	if igm.cm.cluster.Spec.EnableNodePublicIP {
		tpl.Properties.NetworkInterfaces[0].AccessConfigs = []*compute.AccessConfig{
			{
				Name: "Node External IP",
				Type: "ONE_TO_ONE_NAT",
			},
		}
	}
	r1, err := igm.cm.conn.computeService.InstanceTemplates.Insert(igm.cm.cluster.Spec.Project, tpl).Do()
	igm.cm.ctx.Logger().Debug("Create instance template called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.cm.ctx.Logger().Infof("Instance template %v created", templateName)
	return r1.Name, nil
}

func (igm *InstanceGroupManager) createInstanceGroup(sku string, count int64) (string, error) {
	name := igm.cm.namer.InstanceGroupName(sku)
	template := fmt.Sprintf("projects/%v/global/instanceTemplates/%v", igm.cm.cluster.Spec.Project, igm.cm.namer.InstanceTemplateName(sku))

	r1, err := igm.cm.conn.computeService.InstanceGroupManagers.Insert(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, &compute.InstanceGroupManager{
		Name:             name,
		BaseInstanceName: igm.cm.cluster.Name + "-node-" + sku,
		TargetSize:       count,
		InstanceTemplate: template,
	}).Do()
	igm.cm.ctx.Logger().Debug("Create instance group called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.cm.ctx.Logger().Infof("Instance group %v from template %v created", name, template)
	return r1.Name, nil
}

// Not used since Kube 1.3
func (igm *InstanceGroupManager) createAutoscaler(sku string, count int64) (string, error) {
	name := igm.cm.namer.InstanceGroupName(sku)
	target := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%v/zones/%v/instanceGroupManagers/%v", igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, name)

	r1, err := igm.cm.conn.computeService.Autoscalers.Insert(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, &compute.Autoscaler{
		Name:   name,
		Target: target,
		AutoscalingPolicy: &compute.AutoscalingPolicy{
			MinNumReplicas: count,
			MaxNumReplicas: count,
		},
	}).Do()
	igm.cm.ctx.Logger().Debug("Create auto scaler called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.cm.ctx.Logger().Infof("Auto scaler %v for instance group %v is created", name, target)
	return r1.Name, nil
}

func (igm *InstanceGroupManager) deleteOnlyInstanceGroup(instanceGroupName, template string) error {
	_, err := igm.cm.conn.computeService.InstanceGroupManagers.ListManagedInstances(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	r1, err := igm.cm.conn.computeService.InstanceGroupManagers.Delete(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	operation := r1.Name
	igm.cm.conn.waitForZoneOperation(operation)
	igm.cm.ctx.Logger().Infof("Instance group %v is deleted", instanceGroupName)
	igm.cm.ctx.Logger().Infof("Instance template %v is deleting", template)
	r2, err := igm.cm.conn.computeService.InstanceTemplates.Delete(igm.cm.cluster.Spec.Project, template).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	err = igm.cm.conn.waitForGlobalOperation(r2.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.cm.ctx.Logger().Infof("Instance template %v is deleted", template)
	igm.cm.ctx.Logger().Infof("Autoscaler is deleting for %v", instanceGroupName)
	r3, err := igm.cm.conn.computeService.Autoscalers.Delete(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	err = igm.cm.conn.waitForZoneOperation(r3.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.cm.ctx.Logger().Infof("Autoscaler is deleted for %v", instanceGroupName)

	return nil
}

func (igm *InstanceGroupManager) updateInstanceGroup(instanceGroupName string, size int64) error {
	r1, err := igm.cm.conn.computeService.Autoscalers.Get(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	max := r1.AutoscalingPolicy.MaxNumReplicas
	min := r1.AutoscalingPolicy.MinNumReplicas
	igm.cm.ctx.Logger().Infof("Updating autoscaller with Max %v and Min %v num of replicas", size, size)
	if size > max {
		r2, err := igm.cm.conn.computeService.Autoscalers.Patch(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName, &compute.Autoscaler{
			AutoscalingPolicy: &compute.AutoscalingPolicy{
				MaxNumReplicas: size,
				MinNumReplicas: size,
			},
		}).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		err = igm.cm.conn.waitForZoneOperation(r2.Name)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else if size < min {
		r2, err := igm.cm.conn.computeService.Autoscalers.Patch(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName, &compute.Autoscaler{
			AutoscalingPolicy: &compute.AutoscalingPolicy{
				MinNumReplicas: size,
				MaxNumReplicas: size,
			},
		}).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		err = igm.cm.conn.waitForZoneOperation(r2.Name)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	igm.cm.ctx.Logger().Infof("Autoscalling group %v updated", instanceGroupName)
	_, err = igm.cm.conn.computeService.InstanceGroupManagers.ListManagedInstances(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	//sz := int64(len(r.ManagedInstances))
	resp, err := igm.cm.conn.computeService.InstanceGroupManagers.Resize(igm.cm.cluster.Spec.Project, igm.cm.cluster.Spec.Zone, instanceGroupName, size).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.cm.conn.waitForZoneOperation(resp.Name)
	fmt.Println(resp.Name)
	igm.cm.ctx.Logger().Infof("Instance group %v resized", instanceGroupName)
	/*err = cloud.WaitForReadyNodes(igm.cm.ctx, size-sz)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}*/
	// return cluster.Spec.ctx.UpdateNodeCount()
	return nil
}

/*
func DBInstanceManage(ctx *contexts.ClusterContext, instances []*contexts.KubernetesInstance)  {
	kc, err := ctx.NewKubeClient()
	nodes, err := kc.Client.CoreV1().Nodes().List(kapi.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	m := make(map[string]*contexts.KubernetesInstance)
	for _, i := range instances {
		m[i.ExternalID] = i
	}
}*/

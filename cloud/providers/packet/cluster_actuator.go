/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package packet

import (
	"context"

	packet_config "pharmer.dev/pharmer/apis/v1alpha1/packet"

	"k8s.io/client-go/tools/record"
	"k8s.io/klog/klogr"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/cluster"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, func(cm *ClusterManager, m manager.Manager) error {
		actuator := NewClusterActuator(m, ClusterActuatorParams{
			EventRecorder: m.GetEventRecorderFor(Recorder),
			cm:            cm,
		})
		return cluster.AddWithActuator(m, actuator)
	})

}

type ClusterActuator struct {
	client        client.Client
	eventRecorder record.EventRecorder
	cm            *ClusterManager
}

type ClusterActuatorParams struct {
	EventRecorder record.EventRecorder
	cm            *ClusterManager
}

func NewClusterActuator(m manager.Manager, params ClusterActuatorParams) *ClusterActuator {
	params.cm.Logger = klogr.New().WithName("[cachine-actuator]").
		WithValues("cluster-name", params.cm.Cluster.Name)
	return &ClusterActuator{
		client:        m.GetClient(),
		eventRecorder: params.EventRecorder,
		cm:            params.cm,
	}
}
func (ca *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log := ca.cm.Logger

	log.Info("Reconciling cluster")

	if err := packet_config.SetPacketClusterProviderStatus(cluster); err != nil {
		log.Error(err, "Error setting providre status for cluster")
		return err
	}
	return ca.client.Status().Update(context.Background(), cluster)
}

func (ca *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	ca.cm.Logger.Info("Deleting cluster")
	return nil
}

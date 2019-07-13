package packet

import (
	"context"

	"k8s.io/client-go/tools/record"
	"k8s.io/klog/klogr"
	packet_config "pharmer.dev/pharmer/apis/v1beta1/packet"
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

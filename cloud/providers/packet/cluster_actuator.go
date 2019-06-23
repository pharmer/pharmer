package packet

import (
	"context"

	packet_config "github.com/pharmer/pharmer/apis/v1beta1/packet"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
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
			Scheme:        m.GetScheme(),
			cm:            cm,
		})
		return cluster.AddWithActuator(m, actuator)
	})

}

type ClusterActuator struct {
	client        client.Client
	eventRecorder record.EventRecorder
	scheme        *runtime.Scheme
	cm            *ClusterManager
}

type ClusterActuatorParams struct {
	EventRecorder record.EventRecorder
	Scheme        *runtime.Scheme
	cm            *ClusterManager
}

func NewClusterActuator(m manager.Manager, params ClusterActuatorParams) *ClusterActuator {
	return &ClusterActuator{
		client:        m.GetClient(),
		eventRecorder: params.EventRecorder,
		scheme:        params.Scheme,
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

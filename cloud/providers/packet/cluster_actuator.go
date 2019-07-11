package packet

import (
	"context"

	"github.com/appscode/go/log"
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
func (cm *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log.Infof("Reconciling cluster: %q", cluster.Name)

	if err := packet_config.SetPacketClusterProviderStatus(cluster); err != nil {
		log.Debugf("Error setting providre status for cluster %q: %v", cluster.Name, err)
		return err
	}
	return cm.client.Status().Update(context.Background(), cluster)
}

func (cm *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	log.Infof("Deleting cluster %v", cluster.Name)
	return nil
}

package vultr

import (
	"context"
	"fmt"

	"github.com/appscode/go/log"
	vultr_config "github.com/pharmer/pharmer/apis/v1beta1/vultr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/cluster"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, func(ctx context.Context, m manager.Manager, owner string) error {
		actuator := NewClusterActuator(m, ClusterActuatorParams{
			Ctx:           ctx,
			EventRecorder: m.GetEventRecorderFor(Recorder),
			Scheme:        m.GetScheme(),
			Owner:         owner,
		})
		return cluster.AddWithActuator(m, actuator)
	})

}

type ClusterActuator struct {
	ctx           context.Context
	client        client.Client
	eventRecorder record.EventRecorder
	scheme        *runtime.Scheme
	owner         string
}

type ClusterActuatorParams struct {
	Ctx           context.Context
	EventRecorder record.EventRecorder
	Scheme        *runtime.Scheme
	Owner         string
}

func NewClusterActuator(m manager.Manager, params ClusterActuatorParams) *ClusterActuator {
	return &ClusterActuator{
		ctx:           params.Ctx,
		client:        m.GetClient(),
		eventRecorder: params.EventRecorder,
		scheme:        params.Scheme,
		owner:         params.Owner,
	}
}

func (cm *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log.Info("Reconciling cluster: %q", cluster.Name)

	if err := vultr_config.SetVultrClusterProviderStatus(cluster); err != nil {
		log.Debug("Error setting providre status for cluster %q: %v", cluster.Name, err)
		return err
	}
	return cm.client.Status().Update(cm.ctx, cluster)
}

func (cm *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	fmt.Println("Delete cluster %v", cluster.Name)
	return nil
}

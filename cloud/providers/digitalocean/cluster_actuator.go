package digitalocean

import (
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

type Actuator struct {
	client        client.ClusterV1alpha1Interface
	eventRecorder record.EventRecorder
	scheme        *runtime.Scheme
}

type ActuatorParams struct {
	Client        client.ClusterV1alpha1Interface
	EventRecorder record.EventRecorder
	Scheme        *runtime.Scheme
}

func NewActuator(params ActuatorParams) *Actuator {
	return &Actuator{
		client: params.Client,
	}
}

func (cm *ClusterManager) InitializeActuator(client client.ClusterV1alpha1Interface, rec record.EventRecorder, scheme *runtime.Scheme) error {
	cm.actuator = &Actuator{
		client:        client,
		eventRecorder: rec,
		scheme:        scheme,
	}

	return nil
}

func (cm *ClusterManager) Reconcile(cluster *clusterapi.Cluster) error {
	Logger(cm.ctx).Infoln("Reconciling cluster %v", cluster.Name)

	return nil
}

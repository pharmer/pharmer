package linode

import (
	"context"

	linodeApi "github.com/pharmer/pharmer/apis/v1beta1/linode"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
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
	ca.cm.Logger.Info("Reconciling cluster", cluster.Name)

	lbName := ca.cm.namer.LoadBalancerName()
	lb, err := ca.cm.conn.lbByName(lbName)
	if err == errLBNotFound {
		lb, err = ca.cm.conn.createLoadBalancer(lbName)
		if err != nil {
			ca.cm.Logger.Error(err, "error creating load balancer")
			return err
		}
		ca.cm.Logger.Info("created load balancer %q for cluster %q", lbName, ca.cm.conn.Cluster.Name)

		cluster.Status.APIEndpoints = []clusterapi.APIEndpoint{
			{
				Host: *lb.IPv4,
				Port: v1beta1.DefaultAPIBindPort,
			},
		}
	} else if err != nil {
		ca.cm.Logger.Error(err, "error finding load balancer")
		return err
	}

	status, err := linodeApi.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		ca.cm.Logger.Error(err, "Error getting provider status")
		return err
	}
	status.Network.APIServerLB = linodeApi.DescribeLoadBalancer(lb)

	if err := ca.updateClusterStatus(cluster, status); err != nil {
		ca.cm.Logger.Error(err, "Error updating cluster status")
		return err
	}

	ca.cm.Logger.Info("Reconciled cluster successfully")
	return nil
}

func (ca *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	ca.cm.Logger.Info("Delete cluster %v", cluster.Name)
	return nil
}

func (ca *ClusterActuator) updateClusterStatus(cluster *clusterapi.Cluster, status *linodeApi.LinodeClusterProviderStatus) error {
	raw, err := linodeApi.EncodeClusterStatus(status)
	if err != nil {
		ca.cm.Logger.Error(err, "Error encoding cluster status")
		return err
	}

	cluster.Status.ProviderStatus = raw
	return ca.client.Status().Update(context.Background(), cluster)
}

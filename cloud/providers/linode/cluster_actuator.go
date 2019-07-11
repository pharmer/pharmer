package linode

import (
	"context"

	"github.com/go-logr/logr"
	linodeApi "github.com/pharmer/pharmer/apis/v1beta1/linode"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/klogr"
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
	logr.Logger
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
		Logger: klogr.New().WithName("[cluster-actuator]").
			WithValues("cluster-name", params.cm.Cluster.Name),
	}
}

func (ca *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log := ca.Logger
	log.Info("Reconciling cluster")

	lbName := ca.cm.namer.LoadBalancerName()
	lb, err := ca.cm.conn.lbByName(lbName)
	if err == errLBNotFound {
		lb, err = ca.cm.conn.createLoadBalancer(lbName)
		if err != nil {
			log.Error(err, "error creating load balancer")
			return err
		}
		log.Info("created load balancer", "lb-name", lbName)

		cluster.Status.APIEndpoints = []clusterapi.APIEndpoint{
			{
				Host: *lb.IPv4,
				Port: v1beta1.DefaultAPIBindPort,
			},
		}
	} else if err != nil {
		log.Error(err, "error finding load balancer")
		return err
	}

	status, err := linodeApi.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		log.Error(err, "Error getting provider status")
		return err
	}
	status.Network.APIServerLB = linodeApi.DescribeLoadBalancer(lb)

	if err := ca.updateClusterStatus(cluster, status); err != nil {
		log.Error(err, "Error updating cluster status")
		return err
	}

	log.Info("Reconciled cluster successfully")
	return nil
}

func (ca *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	log := ca.Logger
	log.Info("deleting cluster")
	return nil
}

func (ca *ClusterActuator) updateClusterStatus(cluster *clusterapi.Cluster, status *linodeApi.LinodeClusterProviderStatus) error {
	log := ca.Logger
	raw, err := linodeApi.EncodeClusterStatus(status)
	if err != nil {
		log.Error(err, "Error encoding cluster status")
		return err
	}

	cluster.Status.ProviderStatus = raw
	return ca.client.Status().Update(context.Background(), cluster)
}

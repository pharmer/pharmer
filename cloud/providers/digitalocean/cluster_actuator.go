package digitalocean

import (
	"context"

	"github.com/go-logr/logr"
	doCapi "github.com/pharmer/pharmer/apis/v1beta1/digitalocean"
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
			cm:            cm,
		})
		return cluster.AddWithActuator(m, actuator)
	})
}

type ClusterActuator struct {
	client        client.Client
	eventRecorder record.EventRecorder
	cm            *ClusterManager
	logr.Logger
}

type ClusterActuatorParams struct {
	EventRecorder record.EventRecorder
	cm            *ClusterManager
}

func NewClusterActuator(m manager.Manager, params ClusterActuatorParams) *ClusterActuator {
	params.cm.Logger = params.cm.Logger.WithName("[cluster-actuator]")
	return &ClusterActuator{
		client:        m.GetClient(),
		eventRecorder: params.EventRecorder,
		cm:            params.cm,
		Logger: klogr.New().WithName("[cluster-actuator]").
			WithValues("cluster-name", params.cm.Cluster.Name),
	}
}

func (ca *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log := ca.Logger

	log.Info("Reconciling cluster")

	lb, err := ca.cm.conn.lbByName(context.Background(), ca.cm.conn.namer.LoadBalancerName())
	if err == errLBNotFound {
		lb, err = ca.cm.conn.createLoadBalancer(context.Background(), ca.cm.conn.namer.LoadBalancerName())
		if err != nil {
			log.Error(err, "error creating load balancer")
			return err
		}
		log.Info("created load balancer", "lb-name", ca.cm.conn.namer.LoadBalancerName())

		cluster.Status.APIEndpoints = []clusterapi.APIEndpoint{
			{
				Host: lb.IP,
				Port: v1beta1.DefaultAPIBindPort,
			},
		}
	} else if err != nil {
		log.Error(err, "error finding load balancer")
		return err
	}

	updated := ca.cm.conn.loadBalancerUpdated(lb)

	if updated {
		log.Info("Load balancer specs changed, updating lb")

		defaultSpecs := ca.cm.conn.buildLoadBalancerRequest(ca.cm.conn.namer.LoadBalancerName())

		lb, _, err = ca.cm.conn.client.LoadBalancers.Update(context.Background(), lb.ID, defaultSpecs)
		if err != nil {
			log.Error(err, "Error updating load balancer")
			return err
		}
	}

	status, err := doCapi.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		log.Error(err, "Error getting provider status")
		return err
	}
	status.APIServerLB = doCapi.DescribeLoadBalancer(lb)

	if err := ca.updateClusterStatus(cluster, status); err != nil {
		log.Error(err, "Error updating cluster status for cluster")
		return err
	}

	log.Info("Reconciled cluster successfully")
	return nil
}

func (ca *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	ca.Logger.Info("Delete cluster not implemented")

	return nil
}

func (ca *ClusterActuator) updateClusterStatus(cluster *clusterapi.Cluster, status *doCapi.DigitalOceanClusterProviderStatus) error {
	raw, err := doCapi.EncodeClusterStatus(status)
	if err != nil {
		ca.Logger.Error(err, "Error encoding cluster status")
		return err
	}

	cluster.Status.ProviderStatus = raw
	return ca.client.Status().Update(context.Background(), cluster)
}

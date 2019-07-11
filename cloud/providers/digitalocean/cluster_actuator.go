package digitalocean

import (
	"context"

	"github.com/appscode/go/log"
	doCapi "github.com/pharmer/pharmer/apis/v1beta1/digitalocean"
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

func (a *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log.Infoln("Reconciling cluster", cluster.Name)

	lb, err := a.cm.conn.lbByName(context.Background(), a.cm.conn.namer.LoadBalancerName())
	if err == errLBNotFound {
		lb, err = a.cm.conn.createLoadBalancer(context.Background(), a.cm.conn.namer.LoadBalancerName())
		if err != nil {
			log.Debugln("error creating load balancer", err)
			return err
		}
		log.Infof("created load balancer %q for cluster %q", a.cm.conn.namer.LoadBalancerName(), cluster.Name)

		cluster.Status.APIEndpoints = []clusterapi.APIEndpoint{
			{
				Host: lb.IP,
				Port: v1beta1.DefaultAPIBindPort,
			},
		}
	} else if err != nil {
		log.Debugln("error finding load balancer", err)
		return err
	}

	updated := a.cm.conn.loadBalancerUpdated(lb)

	if updated {
		log.Infoln("Load balancer specs changed, updating lb")

		defaultSpecs := a.cm.conn.buildLoadBalancerRequest(a.cm.conn.namer.LoadBalancerName())

		lb, _, err = a.cm.conn.client.LoadBalancers.Update(context.Background(), lb.ID, defaultSpecs)
		if err != nil {
			log.Debugln("Error updating load balancer", err)
			return err
		}
	}

	status, err := doCapi.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		log.Debugln("Error getting provider status", err)
		return err
	}
	status.APIServerLB = doCapi.DescribeLoadBalancer(lb)

	if err := a.updateClusterStatus(cluster, status); err != nil {
		log.Debugf("Error updating cluster status for cluster %q", cluster.Name)
		return err
	}

	log.Infoln("Reconciled cluster successfully")
	return nil
}

func (a *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	log.Infoln("Delete cluster not implemented")

	return nil
}

func (a *ClusterActuator) updateClusterStatus(cluster *clusterapi.Cluster, status *doCapi.DigitalOceanClusterProviderStatus) error {
	raw, err := doCapi.EncodeClusterStatus(status)
	if err != nil {
		log.Debugf("Error encoding cluster status for cluster %q", cluster.Name)
		return err
	}

	cluster.Status.ProviderStatus = raw
	return a.client.Status().Update(context.Background(), cluster)
}

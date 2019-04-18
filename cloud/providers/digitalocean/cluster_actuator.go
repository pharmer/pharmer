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
	AddToManagerFuncs = append(AddToManagerFuncs, func(ctx context.Context, m manager.Manager, owner string) error {
		actuator := NewClusterActuator(m, ClusterActuatorParams{
			Ctx:           ctx,
			EventRecorder: m.GetRecorder(Recorder),
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
	conn          *cloudConnector
}

type ClusterActuatorParams struct {
	Ctx            context.Context
	EventRecorder  record.EventRecorder
	Scheme         *runtime.Scheme
	Owner          string
	CloudConnector *cloudConnector
}

func NewClusterActuator(m manager.Manager, params ClusterActuatorParams) *ClusterActuator {
	return &ClusterActuator{
		ctx:           params.Ctx,
		client:        m.GetClient(),
		eventRecorder: params.EventRecorder,
		scheme:        params.Scheme,
		owner:         params.Owner,
		conn:          params.CloudConnector,
	}
}

func (a *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log.Infoln("Reconciling cluster", cluster.Name)

	conn, err := PrepareCloud(a.ctx, cluster.Name, a.owner)
	if err != nil {
		log.Debugln("Error creating cloud connector", err)
		return err
	}
	a.conn = conn

	// TODO move to reconcileLoadBalance() func if more things are added here
	lb, err := a.conn.lbByName(context.Background(), a.conn.namer.LoadBalancerName())
	if err == errLBNotFound {
		lb, err = a.conn.createLoadBalancer(context.Background(), a.conn.namer.LoadBalancerName())
		if err != nil {
			log.Debugln("error creating load balancer", err)
			return err
		}
		log.Infof("created load balancer %q for cluster %q", a.conn.namer.LoadBalancerName(), cluster.Name)

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

	updated, err := a.conn.loadBalancerUpdated(lb)
	if err != nil {
		return err
	}

	if updated {
		log.Infoln("Load balancer specs changed, updating lb")

		defaultSpecs, err := a.conn.buildLoadBalancerRequest(a.conn.namer.LoadBalancerName())
		if err != nil {
			log.Debugln("Error getting default lb specs")
			return err
		}

		lb, _, err = a.conn.client.LoadBalancers.Update(context.Background(), lb.ID, defaultSpecs)
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
	return a.client.Status().Update(a.ctx, cluster)
}

package linode

import (
	"context"

	"github.com/appscode/go/log"
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

func (cm *ClusterActuator) Reconcile(cluster *clusterapi.Cluster) error {
	log.Infoln("Reconciling cluster", cluster.Name)

	conn, err := PrepareCloud(cm.ctx, cluster.Name, cm.owner)
	if err != nil {
		log.Debugln("Error creating cloud connector", err)
		return err
	}
	cm.conn = conn

	lbName := conn.namer.LoadBalancerName()
	lb, err := conn.lbByName(context.Background(), lbName)
	if err == errLBNotFound {
		lb, err = conn.createLoadBalancer(lbName)
		if err != nil {
			log.Debugln("error creating load balancer", err)
			return err
		}
		log.Infof("created load balancer %q for cluster %q", lbName, conn.cluster.Name)

		cluster.Status.APIEndpoints = []clusterapi.APIEndpoint{
			{
				Host: *lb.IPv4,
				Port: v1beta1.DefaultAPIBindPort,
			},
		}
	} else if err != nil {
		log.Debugln("error finding load balancer", err)
		return err
	}

	status, err := linodeApi.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		log.Debugln("Error getting provider status", err)
		return err
	}
	status.Network.APIServerLB = linodeApi.DescribeLoadBalancer(lb)

	if err := cm.updateClusterStatus(cluster, status); err != nil {
		log.Debugf("Error updating cluster status for cluster %q", cluster.Name)
		return err
	}

	log.Infoln("Reconciled cluster successfully")
	return nil
}

func (cm *ClusterActuator) Delete(cluster *clusterapi.Cluster) error {
	log.Infof("Delete cluster %v", cluster.Name)
	return nil
}

func (cm *ClusterActuator) updateClusterStatus(cluster *clusterapi.Cluster, status *linodeApi.LinodeClusterProviderStatus) error {
	raw, err := linodeApi.EncodeClusterStatus(status)
	if err != nil {
		log.Debugf("Error encoding cluster status for cluster %q", cluster.Name)
		return err
	}

	cluster.Status.ProviderStatus = raw
	return cm.client.Status().Update(cm.ctx, cluster)
}

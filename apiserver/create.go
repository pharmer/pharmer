package apiserver

import (
	"encoding/json"
	"strconv"

	"github.com/nats-io/stan.go"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apiserver/options"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"k8s.io/klog/klogr"
)

func (a *Apiserver) Init(storeProvider store.Interface, msg *stan.Msg) (*api.Operation, *cloud.Scope, error) {
	operation := options.NewClusterOperation()
	err := json.Unmarshal(msg.Data, &operation)
	if err != nil {
		return nil, nil, err
	}
	if operation.OperationId == "" {
		return nil, nil, err
	}

	obj, err := storeProvider.Operations().Get(operation.OperationId)
	if err != nil {
		return nil, nil, err
	}

	// the Cluster().Get() method takes cluster name as parameter
	// if we need to ge cluster usnig ClusterID, then we've to set ownerID as -1
	cluster, err := storeProvider.Owner(-1).Clusters().Get(strconv.Itoa(int(obj.ClusterID)))
	if err != nil {
		return nil, nil, err
	}

	scope := cloud.NewScope(cloud.NewScopeParams{
		Cluster:       cluster,
		StoreProvider: storeProvider.Owner(obj.UserID),
		Logger: klogr.New().WithValues("operationID", obj.ID).
			WithValues("cluster-name", cluster.Name),
	})

	return obj, scope, nil
}

func (a *Apiserver) CreateCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("create-cluster", "cluster-api-create-workers", func(msg *stan.Msg) {
		log := klogr.New().WithName("[apiserver]")

		log.Info("create operation")

		log.V(4).Info("nats message", "sequence", msg.Sequence, "redelivered", msg.Redelivered,
			"message string", string(msg.Data))

		operation, scope, err := a.Init(storeProvider, msg)
		if err != nil {
			log.Error(err, "failed in init")
			return
		}

		log = log.WithValues("operationID", operation.ID)
		log.Info("running operation", "opeartion", operation)

		if operation.State == api.OperationPending {
			operation.State = api.OperationRunning
			operation, err = storeProvider.Operations().Update(operation)
			if err != nil {
				log.Error(err, "failed to update operation", "status", api.OperationRunning)
				return
			}

			if err := msg.Ack(); err != nil {
				log.Error(err, "failed to ACK msg")
				return
			}

			scope.Logger = klogr.New().WithValues("operationID", operation.ID).
				WithValues("cluster-name", scope.Cluster.Name)

			err = cloud.CreateCluster(scope)
			if err != nil {
				log.Error(err, "failed to create cluster")
			}

			scope.Logger = klogr.New().WithValues("operationID", operation.ID).
				WithValues("cluster-name", scope.Cluster.Name)
			err = ApplyCluster(scope, operation)
			if err != nil {
				log.Error(err, "failed to apply cluster")
			}

			log.Info("create operation successfull")
		}
	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

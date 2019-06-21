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

	if obj.State == api.OperationPending {
		obj.State = api.OperationRunning
		obj, err = storeProvider.Operations().Update(obj)
		if err != nil {
			return nil, nil, err
		}

		owner := strconv.Itoa(int(obj.UserID))
		cluster, err := storeProvider.Owner(owner).Clusters().Get(strconv.Itoa(int(obj.ClusterID)))
		if err != nil {
			return nil, nil, err
		}

		scope := cloud.NewScope(cloud.NewScopeParams{
			Cluster:       cluster,
			StoreProvider: storeProvider.Owner(owner),
			Logger: klogr.New().WithName("apiserver").
				WithValues("operation", obj),
		})

		return obj, scope, nil
	}
	return obj, nil, nil
}

func (a *Apiserver) CreateCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("create-cluster", "cluster-api-create-workers", func(msg *stan.Msg) {
		log := klogr.New().WithName("apiserver")
		log.Info("seq", msg.Sequence, "redelivered", msg.Redelivered, "acked", false, "data", msg.Data)

		operation, scope, err := a.Init(storeProvider, msg)
		if err != nil {
			return
		}

		if operation.State == api.OperationPending {
			err = cloud.CreateCluster(scope)
			if err != nil {
				log.Error(err, "failed to create cluster")
			}

			err = ApplyCluster(scope, operation)
			if err != nil {
				log.Error(err, "failed to apply cluster")
			}

			if err := msg.Ack(); err != nil {
				log.Error(err, "failed to ACK msg")
			}
		}
	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

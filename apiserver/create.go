package apiserver

import (
	"encoding/json"
	"errors"
	"strconv"

	stan "github.com/nats-io/stan.go"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apiserver/options"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	natslogr "gomodules.xyz/nats-logr"
	ulogr "gomodules.xyz/union-logr"
)

func (a *Apiserver) Init(storeProvider store.Interface, msg *stan.Msg) (*api.Operation, *cloud.Scope, error) {
	operation := options.NewClusterOperation()
	err := json.Unmarshal(msg.Data, &operation)
	if err != nil {
		return nil, nil, err
	}
	if operation.OperationId == "" {
		return nil, nil, errors.New("operation ID can't be nil")
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
	})

	return obj, scope, nil
}

func (a *Apiserver) CreateCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("create-cluster", "cluster-api-create-workers", func(msg *stan.Msg) {
		operation, scope, err := a.Init(storeProvider, msg)

		opts := natslogr.Options{
			ClusterID: "pharmer-cluster",
			ClientID:  operation.Code,
			NatsURL:   stan.DefaultNatsURL,
			Subject:   scope.Cluster.Name + "-" + strconv.FormatInt(operation.UserID, 10),
		}

		logN := natslogr.NewLogger(opts)
		ulog := ulogr.NewLogger(logN)

		log := ulog.WithName("[apiserver]")

		if err != nil {
			log.Error(err, "failed in init")
			return
		}

		log.Info("create operation")

		log.V(4).Info("nats message", "sequence", msg.Sequence, "redelivered", msg.Redelivered,
			"message string", string(msg.Data))

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

			scope.Logger = ulog.WithValues("operationID", operation.ID).
				WithValues("cluster-name", scope.Cluster.Name)

			err = cloud.CreateCluster(scope)
			if err != nil {
				log.Error(err, "failed to create cluster")
			}

			scope.Logger = ulog.WithValues("operationID", operation.ID).
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

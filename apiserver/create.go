package apiserver

import (
	"github.com/nats-io/stan.go"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/store"
)

func (a *Apiserver) CreateCluster(storeProvider store.Interface, natsurl string, logToNats bool) error {
	_, err := a.natsConn.QueueSubscribe("create-cluster", "cluster-api-create-workers", func(msg *stan.Msg) {
		operation, scope, err := a.Init(storeProvider, msg)

		ulog := newLogger(operation, scope, natsurl, logToNats)
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

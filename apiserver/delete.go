package apiserver

import (
	"github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"k8s.io/klog/klogr"
)

func (a *Apiserver) DeleteCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("delete-cluster", "cluster-api-delete-workers", func(msg *stan.Msg) {
		log := klogr.New().WithName("[apiserver]")

		log.Info("delete operation")
		log.V(4).Info("nats message", "sequence", msg.Sequence, "redelivered", msg.Redelivered,
			"message string", string(msg.Data))

		operation, scope, err := a.Init(storeProvider, msg)
		if err != nil {
			log.Error(err, "failed init func")
			return
		}

		log = log.WithValues("operationID", operation.ID)
		log.Info("running operation", "operation", operation)

		cluster, err := cloud.Delete(scope.StoreProvider.Clusters(), scope.Cluster.Name)
		if err != nil {
			log.Error(err, "failed to delete cluster")
			return
		}

		scope.Cluster = cluster

		err = ApplyCluster(scope, operation)
		if err != nil {
			log.Error(err, "failed to apply cluster delete operation")
			return
		}

		if err := msg.Ack(); err != nil {
			log.Error(err, "failed to ACK msg")
			return
		}
		log.Info("delete operation success")

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

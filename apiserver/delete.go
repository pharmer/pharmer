package apiserver

import (
	"github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
)

func (a *Apiserver) DeleteCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("delete-cluster", "cluster-api-delete-workers", func(msg *stan.Msg) {
		log := klogr.New().WithName("apiserver")
		log.Info("seq", "sequence", msg.Sequence, "redelivered", msg.Redelivered, "acked", false, "data", string(msg.Data))

		log.Info("delete operation")

		operation, scope, err := a.Init(storeProvider, msg)
		if err != nil {
			log.Error(err, "failed init func")
			return
		}

		cluster, err := cloud.Delete(scope.StoreProvider.Clusters(), scope.Cluster.Name)
		if err != nil {
			log.Error(err, "failed to delete cluster")
		}

		scope.Cluster = cluster

		err = ApplyCluster(scope, operation)
		if err != nil {
			log.Error(err, "failed to apply cluster delete operation")
		}

		if err := msg.Ack(); err != nil {
			klog.Errorf("failed to ACK msg: %d", msg.Sequence)
		}
		log.Info("delete operation success")

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

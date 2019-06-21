package apiserver

import (
	"github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/store"
	"k8s.io/klog/klogr"
)

func (a *Apiserver) RetryCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("retry-cluster", "cluster-api-retry-workers", func(msg *stan.Msg) {
		log := klogr.New().WithName("apiserver")
		log.Info("seq", msg.Sequence, "redelivered", msg.Redelivered, "acked", false, "data", msg.Data)

		operation, scope, err := a.Init(storeProvider, msg)
		if err != nil {
			log.Error(err, "failed init func")
			return
		}

		err = ApplyCluster(scope, operation)
		if err != nil {
			log.Error(err, "failed to apply cluster")
		}

		if err := msg.Ack(); err != nil {
			log.Error(err, "failed to ack msg")
		}

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

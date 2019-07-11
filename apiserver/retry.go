package apiserver

import (
	"github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/store"
	"k8s.io/klog/klogr"
)

func (a *Apiserver) RetryCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("retry-cluster", "cluster-api-retry-workers", func(msg *stan.Msg) {
		log := klogr.New().WithName("[apiserver]")

		log.Info("retry operation")

		log.V(4).Info("nats message", "sequence", msg.Sequence, "redelivered", msg.Redelivered,
			"message string", string(msg.Data))

		operation, scope, err := a.Init(storeProvider, msg)
		if err != nil {
			log.Error(err, "failed init func")
			return
		}

		log = log.WithValues("operationID", operation.ID)
		log.Info("running operation", "opeartion", operation)

		if err := msg.Ack(); err != nil {
			log.Error(err, "failed to ack msg")
			return
		}

		err = ApplyCluster(scope, operation)
		if err != nil {
			log.Error(err, "failed to apply cluster")
			return
		}

		log.Info("retry operation successfull")

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

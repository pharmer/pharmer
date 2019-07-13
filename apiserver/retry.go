package apiserver

import (
	"github.com/nats-io/stan.go"
	"pharmer.dev/pharmer/store"
)

func (a *Apiserver) RetryCluster(storeProvider store.Interface, natsurl string, logToNats bool) error {
	_, err := a.natsConn.QueueSubscribe("retry-cluster", "cluster-api-retry-workers", func(msg *stan.Msg) {
		operation, scope, err := a.Init(storeProvider, msg)

		ulog := newLogger(operation, scope, natsurl, logToNats)
		log := ulog.WithName("[apiserver]")

		if err != nil {
			log.Error(err, "failed in init")
			return
		}

		log.Info("retry operation")

		log.V(4).Info("nats message", "sequence", msg.Sequence, "redelivered", msg.Redelivered,
			"message string", string(msg.Data))

		log = log.WithValues("operationID", operation.ID)
		log.Info("running operation", "opeartion", operation)

		if err := msg.Ack(); err != nil {
			log.Error(err, "failed to ack msg")
			return
		}

		scope.Logger = ulog.WithValues("operationID", operation.ID).
			WithValues("cluster-name", scope.Cluster.Name)

		err = ApplyCluster(scope, operation)
		if err != nil {
			log.Error(err, "failed to apply cluster")
			return
		}

		log.Info("retry operation successfull")

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

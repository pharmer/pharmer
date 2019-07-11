package apiserver

import (
	"strconv"

	stan "github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	natslogr "gomodules.xyz/nats-logr"
	ulogr "gomodules.xyz/union-logr"
)

func (a *Apiserver) DeleteCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("delete-cluster", "cluster-api-delete-workers", func(msg *stan.Msg) {
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

		log.Info("delete operation")
		log.V(4).Info("nats message", "sequence", msg.Sequence, "redelivered", msg.Redelivered,
			"message string", string(msg.Data))

		log = log.WithValues("operationID", operation.ID)
		log.Info("running operation", "operation", operation)

		cluster, err := cloud.Delete(scope.StoreProvider.Clusters(), scope.Cluster.Name)
		if err != nil {
			log.Error(err, "failed to delete cluster")
			return
		}

		if err := msg.Ack(); err != nil {
			log.Error(err, "failed to ACK msg")
			return
		}

		scope.Cluster = cluster
		scope.Logger = ulog.WithValues("operationID", operation.ID).
			WithValues("cluster-name", scope.Cluster.Name)

		err = ApplyCluster(scope, operation)
		if err != nil {
			log.Error(err, "failed to apply cluster delete operation")
			return
		}

		log.Info("delete operation success")

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

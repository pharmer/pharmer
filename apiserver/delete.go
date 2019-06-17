package apiserver

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/nats-io/stan.go"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apiserver/options"
	"github.com/pharmer/pharmer/cloud"
	opts "github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
)

func (a *Apiserver) DeleteCluster() error {
	_, err := a.natsConn.QueueSubscribe("delete-cluster", "cluster-api-delete-workers", func(msg *stan.Msg) {
		fmt.Printf("seq = %d [redelivered = %v, acked = false]\n", msg.Sequence, msg.Redelivered)

		operation := options.NewClusterOperation()
		err := json.Unmarshal(msg.Data, &operation)
		if err != nil {
			glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			return
		}
		if operation.OperationId == "" {
			err := fmt.Errorf("operation id not  found")
			glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			return
		}
		obj, err := store.StoreProvider.Operations().Get(operation.OperationId)
		if err != nil {
			glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
		}

		clusterID := strconv.Itoa(int(obj.ClusterID))

		if obj.State == api.OperationPending {
			obj.State = api.OperationRunning
			obj, err = store.StoreProvider.Operations().Update(obj)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			cluster, err := store.StoreProvider.Clusters().Get(clusterID)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			cluster, err = cloud.Delete(cluster.Name)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			ApplyCluster(&opts.ApplyConfig{
				ClusterName: cluster.Name, //strconv.Itoa(int(obj.ClusterID)),
				Owner:       strconv.Itoa(int(obj.UserID)),
				DryRun:      false,
			}, obj)

			if err := msg.Ack(); err != nil {
				glog.Errorf("failed to ACK msg: %d", msg.Sequence)
			}

		}

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

package apiserver

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	stan "github.com/nats-io/go-nats-streaming"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apiserver/options"
	. "github.com/pharmer/pharmer/cloud"
	opts "github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/notification"
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
			err := fmt.Errorf("Operation id not  found")
			glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			return
		}
		obj, err := Store(a.ctx).Operations().Get(operation.OperationId)
		if err != nil {
			glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
		}

		clusterID := strconv.Itoa(int(obj.ClusterID))

		if obj.State == api.OperationPending {
			obj.State = api.OperationRunning
			obj, err = Store(a.ctx).Operations().Update(obj)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			cluster, err := Store(a.ctx).Clusters().Get(clusterID)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			noti := notification.NewNotifier(a.ctx, a.natsConn, strconv.Itoa(int(obj.ClusterID)))
			newCtx := WithLogger(a.ctx, noti)

			cluster, err = Delete(newCtx, cluster.Name, strconv.Itoa(int(obj.UserID)))
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			ApplyCluster(newCtx, &opts.ApplyConfig{
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

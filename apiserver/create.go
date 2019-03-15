package apiserver

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/nats-io/go-nats-streaming"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apiserver/options"
	. "github.com/pharmer/pharmer/cloud"
	opts "github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/notification"
	"strconv"
	"time"
)

func (a *Apiserver) CreateCluster() error {
	_, err := a.natsConn.QueueSubscribe("create-cluster", "cluster-api-create-workers", func(msg *stan.Msg) {
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

		if obj.State == api.OperationPending {
			obj.State = api.OperationRunning
			obj, err = Store(a.ctx).Operations().Update(obj)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			cluster, err := Store(a.ctx).Clusters().Get(strconv.Itoa(int(obj.ClusterID)))
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			cluster.InitClusterApi()

			noti := notification.NewNotifier(a.ctx, a.natsConn, strconv.Itoa(int(obj.ClusterID)))
			newCtx := WithLogger(a.ctx, noti)


			cluster, err = Create(newCtx, cluster, strconv.Itoa(int(obj.UserID)))
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


	}, stan.SetManualAckMode(), stan.AckWait(time.Second))

	return err
}

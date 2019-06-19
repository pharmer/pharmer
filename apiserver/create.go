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

func (a *Apiserver) CreateCluster(storeProvider store.Interface) error {
	_, err := a.natsConn.QueueSubscribe("create-cluster", "cluster-api-create-workers", func(msg *stan.Msg) {
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

		obj, err := storeProvider.Operations().Get(operation.OperationId)
		if err != nil {
			glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
		}

		if obj.State == api.OperationPending {
			obj.State = api.OperationRunning
			obj, err = storeProvider.Operations().Update(obj)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			owner := strconv.Itoa(int(obj.UserID))
			cluster, err := storeProvider.Owner(owner).Clusters().Get(strconv.Itoa(int(obj.ClusterID)))
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			cluster.InitClusterAPI()

			err = cloud.CreateCluster(storeProvider, cluster)
			if err != nil {
				glog.Errorf("seq = %d [redelivered = %v, data = %v, err = %v]\n", msg.Sequence, msg.Redelivered, msg.Data, err)
			}

			ApplyCluster(storeProvider, &opts.ApplyConfig{
				ClusterName: cluster.Name, //strconv.Itoa(int(obj.ClusterID)),
				Owner:       owner,
				DryRun:      false,
			}, obj)

			if err := msg.Ack(); err != nil {
				glog.Errorf("failed to ACK msg: %d", msg.Sequence)
			}

		}

	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package apiserver

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/store"

	"github.com/nats-io/stan.go"
)

func (a *Apiserver) CreateCluster(storeProvider store.Interface, natsurl string, logToNats bool) error {
	_, err := a.natsConn.QueueSubscribe("create-cluster", "cluster-api-create-workers", func(msg *stan.Msg) {
		operation, scope, err := a.Init(storeProvider, msg)

		ulog := newLogger(operation, scope, natsurl, logToNats)
		log := ulog.WithName("[apiserver]")

		if err != nil {
			log.Error(err, "failed in init")
			return
		}

		log.Info("create operation")

		log.V(4).Info("nats message", "sequence", msg.Sequence, "redelivered", msg.Redelivered,
			"message string", string(msg.Data))

		log = log.WithValues("operationID", operation.ID)
		log.Info("running operation", "opeartion", operation)

		if operation.State == api.OperationPending {
			operation.State = api.OperationRunning
			operation, err = storeProvider.Operations().Update(operation)
			if err != nil {
				log.Error(err, "failed to update operation", "status", api.OperationRunning)
				return
			}

			if err := msg.Ack(); err != nil {
				log.Error(err, "failed to ACK msg")
				return
			}

			scope.Logger = ulog.WithValues("operationID", operation.ID).
				WithValues("cluster-name", scope.Cluster.Name)

			err = cloud.CreateCluster(scope)
			if err != nil {
				log.Error(err, "failed to create cluster")
			}

			scope.Logger = ulog.WithValues("operationID", operation.ID).
				WithValues("cluster-name", scope.Cluster.Name)
			err = ApplyCluster(scope, operation)
			if err != nil {
				log.Error(err, "failed to apply cluster")
			}

			log.Info("create operation successfull")
		}
	}, stan.SetManualAckMode(), stan.DurableName("i-remember"))

	return err
}

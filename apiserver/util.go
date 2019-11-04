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
	"encoding/json"
	"errors"
	"strconv"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/apiserver/options"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/store"

	"github.com/go-logr/logr"
	"github.com/nats-io/stan.go"
	natslogr "gomodules.xyz/nats-logr"
	ulogr "gomodules.xyz/union-logr"
	"k8s.io/klog/klogr"
)

func (a *Apiserver) Init(storeProvider store.Interface, msg *stan.Msg) (*api.Operation, *cloud.Scope, error) {
	operation := options.NewClusterOperation()
	err := json.Unmarshal(msg.Data, &operation)
	if err != nil {
		return nil, nil, err
	}
	if operation.OperationId == "" {
		return nil, nil, errors.New("operation ID can't be nil")
	}

	obj, err := storeProvider.Operations().Get(operation.OperationId)
	if err != nil {
		return nil, nil, err
	}

	// the Cluster().Get() method takes cluster name as parameter
	// if we need to ge cluster usnig ClusterID, then we've to set ownerID as -1
	cluster, err := storeProvider.Owner(-1).Clusters().Get(strconv.Itoa(int(obj.ClusterID)))
	if err != nil {
		return nil, nil, err
	}

	scope := cloud.NewScope(cloud.NewScopeParams{
		Cluster:       cluster,
		StoreProvider: storeProvider.Owner(obj.UserID),
	})

	return obj, scope, nil
}

func newLogger(operation *api.Operation, scope *cloud.Scope, natsurl string, logToNats bool) logr.Logger {
	logK := klogr.New()
	if !logToNats {
		return ulogr.NewLogger(logK)
	}

	opts := natslogr.Options{
		ClusterID: "pharmer-cluster",
		ClientID:  operation.Code,
		NatsURL:   natsurl,
		Subject:   scope.Cluster.Name + "-" + strconv.FormatInt(operation.UserID, 10),
	}
	logN := natslogr.NewLogger(opts)

	return ulogr.NewLogger(logK, logN)
}

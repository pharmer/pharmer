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
package v1alpha1

import (
	"strings"

	cloudapi "pharmer.dev/cloud/apis/cloud/v1"

	"github.com/pkg/errors"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	clusterAPIversion = "cluster.k8s.io/v1alpha1"
	pharmerAPIversion = "cluster.pharmer.io/v1beta1"
)

func AssignTypeKind(v interface{}) error {
	switch u := v.(type) {
	case *PharmerConfig:
		if u.APIVersion == "" {
			u.APIVersion = pharmerAPIversion
		}
		u.Kind = "PharmerConfig"
		return nil
	case *Cluster:
		if u.APIVersion == "" {
			u.APIVersion = pharmerAPIversion
		}
		u.Kind = "Cluster"
		return nil
	case *cloudapi.Credential:
		if u.APIVersion == "" {
			u.APIVersion = "cloud.pharmer.io/v1"
		}
		u.Kind = "Credential"
		return nil
	case *clusterapi.Cluster:
		if u.APIVersion == "" {
			u.APIVersion = clusterAPIversion
		}
		u.Kind = "Cluster"
		return nil
	case *clusterapi.Machine:
		if u.APIVersion == "" {
			u.APIVersion = clusterAPIversion
		}
		u.Kind = "Machine"
		return nil
	case *clusterapi.MachineSet:
		if u.APIVersion == "" {
			u.APIVersion = clusterAPIversion
		}
		u.Kind = "MachineSet"
		return nil
	}
	return errors.New("Unknown api object type")
}

// ErrAlreadyExist checks if the error occurred due to the object already present in the cluster
func ErrAlreadyExist(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}

// ErrObjectModified checks if the error is :
// Operation cannot be fulfilled on machines.cluster.k8s.io <machine>:
// the object has been modified; please apply your changes to the latest version and try again
func ErrObjectModified(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "the object has been modified")
}

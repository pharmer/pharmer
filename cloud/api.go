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
package cloud

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	ErrNotFound       = errors.New("node not found")
	ErrNotImplemented = errors.New("not implemented")
	ErrNoMasterNG     = errors.New("Cluster has no master NodeGroup")
)

type Interface interface {
	CloudManagerInterface

	CreateCredentials(kc kubernetes.Interface) error

	AddToManager(m manager.Manager) error

	ProviderKubeConfig

	SetCloudConnector() error

	ApplyDelete() error
	// only managed providers
	ApplyScale() error
	SetDefaultCluster() error
	GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error)

	NewMasterTemplateData(machine *clusterapi.Machine, token string, td TemplateData) TemplateData
	NewNodeTemplateData(machine *clusterapi.Machine, token string, td TemplateData) TemplateData

	PrepareCloud() error
	EnsureMaster(machine *clusterapi.Machine) error

	GetMasterSKU(totalNodes int32) string

	GetClusterAPIComponents() (string, error)
}

type UpgradeManager interface {
	GetAvailableUpgrades() ([]*api.Upgrade, error)
	PrintAvailableUpgrades([]*api.Upgrade)
	Apply() error
	MasterUpgrade(oldMachine *clusterapi.Machine, newMachine *clusterapi.Machine) error
	NodeUpgrade(oldMachine *clusterapi.Machine, newMachine *clusterapi.Machine) error
}

type ProviderKubeConfig interface {
	GetKubeConfig() (*api.KubeConfig, error)
}

type HookFunc func() error

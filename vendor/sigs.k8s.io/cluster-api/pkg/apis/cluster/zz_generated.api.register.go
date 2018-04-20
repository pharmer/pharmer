/*
Copyright 2017 The Kubernetes Authors.

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

// This file was autogenerated by apiregister-gen. Do not edit it manually!

package cluster

import (
	"fmt"
	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilintstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	clustercommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
)

var (
	InternalCluster = builders.NewInternalResource(
		"clusters",
		"Cluster",
		func() runtime.Object { return &Cluster{} },
		func() runtime.Object { return &ClusterList{} },
	)
	InternalClusterStatus = builders.NewInternalResourceStatus(
		"clusters",
		"ClusterStatus",
		func() runtime.Object { return &Cluster{} },
		func() runtime.Object { return &ClusterList{} },
	)
	InternalMachine = builders.NewInternalResource(
		"machines",
		"Machine",
		func() runtime.Object { return &Machine{} },
		func() runtime.Object { return &MachineList{} },
	)
	InternalMachineStatus = builders.NewInternalResourceStatus(
		"machines",
		"MachineStatus",
		func() runtime.Object { return &Machine{} },
		func() runtime.Object { return &MachineList{} },
	)
	InternalMachineDeployment = builders.NewInternalResource(
		"machinedeployments",
		"MachineDeployment",
		func() runtime.Object { return &MachineDeployment{} },
		func() runtime.Object { return &MachineDeploymentList{} },
	)
	InternalMachineDeploymentStatus = builders.NewInternalResourceStatus(
		"machinedeployments",
		"MachineDeploymentStatus",
		func() runtime.Object { return &MachineDeployment{} },
		func() runtime.Object { return &MachineDeploymentList{} },
	)
	InternalMachineSet = builders.NewInternalResource(
		"machinesets",
		"MachineSet",
		func() runtime.Object { return &MachineSet{} },
		func() runtime.Object { return &MachineSetList{} },
	)
	InternalMachineSetStatus = builders.NewInternalResourceStatus(
		"machinesets",
		"MachineSetStatus",
		func() runtime.Object { return &MachineSet{} },
		func() runtime.Object { return &MachineSetList{} },
	)
	// Registered resources and subresources
	ApiVersion = builders.NewApiGroup("cluster.k8s.io").WithKinds(
		InternalCluster,
		InternalClusterStatus,
		InternalMachine,
		InternalMachineStatus,
		InternalMachineDeployment,
		InternalMachineDeploymentStatus,
		InternalMachineSet,
		InternalMachineSetStatus,
	)

	// Required by code generated by go2idl
	AddToScheme        = ApiVersion.SchemaBuilder.AddToScheme
	SchemeBuilder      = ApiVersion.SchemaBuilder
	localSchemeBuilder = &SchemeBuilder
	SchemeGroupVersion = ApiVersion.GroupVersion
)

// Required by code generated by go2idl
// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Required by code generated by go2idl
// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// +genclient
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Machine struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   MachineSpec
	Status MachineStatus
}

// +genclient
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Cluster struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   ClusterSpec
	Status ClusterStatus
}

type MachineStatus struct {
	NodeRef      *corev1.ObjectReference
	LastUpdated  metav1.Time
	Versions     *MachineVersionInfo
	ErrorReason  *clustercommon.MachineStatusError
	ErrorMessage *string
}

type ClusterStatus struct {
	APIEndpoints   []APIEndpoint
	ErrorReason    clustercommon.ClusterStatusError
	ErrorMessage   string
	ProviderStatus string
}

type MachineVersionInfo struct {
	Kubelet          string
	ControlPlane     string
	ContainerRuntime ContainerRuntimeInfo
}

type APIEndpoint struct {
	Host string
	Port int
}

type ContainerRuntimeInfo struct {
	Name    string
	Version string
}

type ClusterSpec struct {
	ClusterNetwork ClusterNetworkingConfig
	ProviderConfig ProviderConfig
}

type MachineSpec struct {
	metav1.ObjectMeta
	Taints         []corev1.Taint
	ProviderConfig ProviderConfig
	Roles          []clustercommon.MachineRole
	Versions       MachineVersionInfo
	ConfigSource   *corev1.NodeConfigSource
}

type ProviderConfig struct {
	Value     *pkgruntime.RawExtension
	ValueFrom *ProviderConfigSource
}

type ProviderConfigSource struct {
}

type ClusterNetworkingConfig struct {
	Services      NetworkRanges
	Pods          NetworkRanges
	ServiceDomain string
}

// +genclient
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineSet struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   MachineSetSpec
	Status MachineSetStatus
}

type NetworkRanges struct {
	CIDRBlocks []string
}

type MachineSetStatus struct {
	Replicas             int32
	FullyLabeledReplicas int32
	ReadyReplicas        int32
	AvailableReplicas    int32
	ObservedGeneration   int64
	ErrorReason          *clustercommon.MachineSetStatusError
	ErrorMessage         *string
}

type MachineSetSpec struct {
	Replicas        *int32
	MinReadySeconds int32
	Selector        metav1.LabelSelector
	Template        MachineTemplateSpec
}

type MachineTemplateSpec struct {
	metav1.ObjectMeta
	Spec MachineSpec
}

// +genclient
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineDeployment struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec   MachineDeploymentSpec
	Status MachineDeploymentStatus
}

type MachineDeploymentStatus struct {
	ObservedGeneration  int64
	Replicas            int32
	UpdatedReplicas     int32
	ReadyReplicas       int32
	AvailableReplicas   int32
	UnavailableReplicas int32
}

type MachineDeploymentSpec struct {
	Replicas                *int32
	Selector                metav1.LabelSelector
	Template                MachineTemplateSpec
	Strategy                MachineDeploymentStrategy
	MinReadySeconds         *int32
	RevisionHistoryLimit    *int32
	Paused                  bool
	ProgressDeadlineSeconds *int32
}

type MachineDeploymentStrategy struct {
	Type          clustercommon.MachineDeploymentStrategyType
	RollingUpdate *MachineRollingUpdateDeployment
}

type MachineRollingUpdateDeployment struct {
	MaxUnavailable *utilintstr.IntOrString
	MaxSurge       *utilintstr.IntOrString
}

//
// Cluster Functions and Structs
//
// +k8s:deepcopy-gen=false
type ClusterStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type ClusterStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Cluster
}

func (Cluster) NewStatus() interface{} {
	return ClusterStatus{}
}

func (pc *Cluster) GetStatus() interface{} {
	return pc.Status
}

func (pc *Cluster) SetStatus(s interface{}) {
	pc.Status = s.(ClusterStatus)
}

func (pc *Cluster) GetSpec() interface{} {
	return pc.Spec
}

func (pc *Cluster) SetSpec(s interface{}) {
	pc.Spec = s.(ClusterSpec)
}

func (pc *Cluster) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

func (pc *Cluster) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

func (pc Cluster) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}

// Registry is an interface for things that know how to store Cluster.
// +k8s:deepcopy-gen=false
type ClusterRegistry interface {
	ListClusters(ctx request.Context, options *internalversion.ListOptions) (*ClusterList, error)
	GetCluster(ctx request.Context, id string, options *metav1.GetOptions) (*Cluster, error)
	CreateCluster(ctx request.Context, id *Cluster) (*Cluster, error)
	UpdateCluster(ctx request.Context, id *Cluster) (*Cluster, error)
	DeleteCluster(ctx request.Context, id string) (bool, error)
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched types will panic.
func NewClusterRegistry(sp builders.StandardStorageProvider) ClusterRegistry {
	return &storageCluster{sp}
}

// Implement Registry
// storage puts strong typing around storage calls
// +k8s:deepcopy-gen=false
type storageCluster struct {
	builders.StandardStorageProvider
}

func (s *storageCluster) ListClusters(ctx request.Context, options *internalversion.ListOptions) (*ClusterList, error) {
	if options != nil && options.FieldSelector != nil && !options.FieldSelector.Empty() {
		return nil, fmt.Errorf("field selector not supported yet")
	}
	st := s.GetStandardStorage()
	obj, err := st.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*ClusterList), err
}

func (s *storageCluster) GetCluster(ctx request.Context, id string, options *metav1.GetOptions) (*Cluster, error) {
	st := s.GetStandardStorage()
	obj, err := st.Get(ctx, id, options)
	if err != nil {
		return nil, err
	}
	return obj.(*Cluster), nil
}

func (s *storageCluster) CreateCluster(ctx request.Context, object *Cluster) (*Cluster, error) {
	st := s.GetStandardStorage()
	obj, err := st.Create(ctx, object, nil, true)
	if err != nil {
		return nil, err
	}
	return obj.(*Cluster), nil
}

func (s *storageCluster) UpdateCluster(ctx request.Context, object *Cluster) (*Cluster, error) {
	st := s.GetStandardStorage()
	obj, _, err := st.Update(ctx, object.Name, rest.DefaultUpdatedObjectInfo(object), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*Cluster), nil
}

func (s *storageCluster) DeleteCluster(ctx request.Context, id string) (bool, error) {
	st := s.GetStandardStorage()
	_, sync, err := st.Delete(ctx, id, nil)
	return sync, err
}

//
// Machine Functions and Structs
//
// +k8s:deepcopy-gen=false
type MachineStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type MachineStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Machine
}

func (Machine) NewStatus() interface{} {
	return MachineStatus{}
}

func (pc *Machine) GetStatus() interface{} {
	return pc.Status
}

func (pc *Machine) SetStatus(s interface{}) {
	pc.Status = s.(MachineStatus)
}

func (pc *Machine) GetSpec() interface{} {
	return pc.Spec
}

func (pc *Machine) SetSpec(s interface{}) {
	pc.Spec = s.(MachineSpec)
}

func (pc *Machine) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

func (pc *Machine) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

func (pc Machine) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}

// Registry is an interface for things that know how to store Machine.
// +k8s:deepcopy-gen=false
type MachineRegistry interface {
	ListMachines(ctx request.Context, options *internalversion.ListOptions) (*MachineList, error)
	GetMachine(ctx request.Context, id string, options *metav1.GetOptions) (*Machine, error)
	CreateMachine(ctx request.Context, id *Machine) (*Machine, error)
	UpdateMachine(ctx request.Context, id *Machine) (*Machine, error)
	DeleteMachine(ctx request.Context, id string) (bool, error)
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched types will panic.
func NewMachineRegistry(sp builders.StandardStorageProvider) MachineRegistry {
	return &storageMachine{sp}
}

// Implement Registry
// storage puts strong typing around storage calls
// +k8s:deepcopy-gen=false
type storageMachine struct {
	builders.StandardStorageProvider
}

func (s *storageMachine) ListMachines(ctx request.Context, options *internalversion.ListOptions) (*MachineList, error) {
	if options != nil && options.FieldSelector != nil && !options.FieldSelector.Empty() {
		return nil, fmt.Errorf("field selector not supported yet")
	}
	st := s.GetStandardStorage()
	obj, err := st.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineList), err
}

func (s *storageMachine) GetMachine(ctx request.Context, id string, options *metav1.GetOptions) (*Machine, error) {
	st := s.GetStandardStorage()
	obj, err := st.Get(ctx, id, options)
	if err != nil {
		return nil, err
	}
	return obj.(*Machine), nil
}

func (s *storageMachine) CreateMachine(ctx request.Context, object *Machine) (*Machine, error) {
	st := s.GetStandardStorage()
	obj, err := st.Create(ctx, object, nil, true)
	if err != nil {
		return nil, err
	}
	return obj.(*Machine), nil
}

func (s *storageMachine) UpdateMachine(ctx request.Context, object *Machine) (*Machine, error) {
	st := s.GetStandardStorage()
	obj, _, err := st.Update(ctx, object.Name, rest.DefaultUpdatedObjectInfo(object), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*Machine), nil
}

func (s *storageMachine) DeleteMachine(ctx request.Context, id string) (bool, error) {
	st := s.GetStandardStorage()
	_, sync, err := st.Delete(ctx, id, nil)
	return sync, err
}

//
// MachineDeployment Functions and Structs
//
// +k8s:deepcopy-gen=false
type MachineDeploymentValidationStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type MachineDeploymentValidationStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineDeploymentList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []MachineDeployment
}

func (MachineDeployment) NewStatus() interface{} {
	return MachineDeploymentStatus{}
}

func (pc *MachineDeployment) GetStatus() interface{} {
	return pc.Status
}

func (pc *MachineDeployment) SetStatus(s interface{}) {
	pc.Status = s.(MachineDeploymentStatus)
}

func (pc *MachineDeployment) GetSpec() interface{} {
	return pc.Spec
}

func (pc *MachineDeployment) SetSpec(s interface{}) {
	pc.Spec = s.(MachineDeploymentSpec)
}

func (pc *MachineDeployment) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

func (pc *MachineDeployment) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

func (pc MachineDeployment) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}

// Registry is an interface for things that know how to store MachineDeployment.
// +k8s:deepcopy-gen=false
type MachineDeploymentRegistry interface {
	ListMachineDeployments(ctx request.Context, options *internalversion.ListOptions) (*MachineDeploymentList, error)
	GetMachineDeployment(ctx request.Context, id string, options *metav1.GetOptions) (*MachineDeployment, error)
	CreateMachineDeployment(ctx request.Context, id *MachineDeployment) (*MachineDeployment, error)
	UpdateMachineDeployment(ctx request.Context, id *MachineDeployment) (*MachineDeployment, error)
	DeleteMachineDeployment(ctx request.Context, id string) (bool, error)
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched types will panic.
func NewMachineDeploymentRegistry(sp builders.StandardStorageProvider) MachineDeploymentRegistry {
	return &storageMachineDeployment{sp}
}

// Implement Registry
// storage puts strong typing around storage calls
// +k8s:deepcopy-gen=false
type storageMachineDeployment struct {
	builders.StandardStorageProvider
}

func (s *storageMachineDeployment) ListMachineDeployments(ctx request.Context, options *internalversion.ListOptions) (*MachineDeploymentList, error) {
	if options != nil && options.FieldSelector != nil && !options.FieldSelector.Empty() {
		return nil, fmt.Errorf("field selector not supported yet")
	}
	st := s.GetStandardStorage()
	obj, err := st.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineDeploymentList), err
}

func (s *storageMachineDeployment) GetMachineDeployment(ctx request.Context, id string, options *metav1.GetOptions) (*MachineDeployment, error) {
	st := s.GetStandardStorage()
	obj, err := st.Get(ctx, id, options)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineDeployment), nil
}

func (s *storageMachineDeployment) CreateMachineDeployment(ctx request.Context, object *MachineDeployment) (*MachineDeployment, error) {
	st := s.GetStandardStorage()
	obj, err := st.Create(ctx, object, nil, true)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineDeployment), nil
}

func (s *storageMachineDeployment) UpdateMachineDeployment(ctx request.Context, object *MachineDeployment) (*MachineDeployment, error) {
	st := s.GetStandardStorage()
	obj, _, err := st.Update(ctx, object.Name, rest.DefaultUpdatedObjectInfo(object), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineDeployment), nil
}

func (s *storageMachineDeployment) DeleteMachineDeployment(ctx request.Context, id string) (bool, error) {
	st := s.GetStandardStorage()
	_, sync, err := st.Delete(ctx, id, nil)
	return sync, err
}

//
// MachineSet Functions and Structs
//
// +k8s:deepcopy-gen=false
type MachineSetStrategy struct {
	builders.DefaultStorageStrategy
}

// +k8s:deepcopy-gen=false
type MachineSetStatusStrategy struct {
	builders.DefaultStatusStorageStrategy
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineSetList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []MachineSet
}

func (MachineSet) NewStatus() interface{} {
	return MachineSetStatus{}
}

func (pc *MachineSet) GetStatus() interface{} {
	return pc.Status
}

func (pc *MachineSet) SetStatus(s interface{}) {
	pc.Status = s.(MachineSetStatus)
}

func (pc *MachineSet) GetSpec() interface{} {
	return pc.Spec
}

func (pc *MachineSet) SetSpec(s interface{}) {
	pc.Spec = s.(MachineSetSpec)
}

func (pc *MachineSet) GetObjectMeta() *metav1.ObjectMeta {
	return &pc.ObjectMeta
}

func (pc *MachineSet) SetGeneration(generation int64) {
	pc.ObjectMeta.Generation = generation
}

func (pc MachineSet) GetGeneration() int64 {
	return pc.ObjectMeta.Generation
}

// Registry is an interface for things that know how to store MachineSet.
// +k8s:deepcopy-gen=false
type MachineSetRegistry interface {
	ListMachineSets(ctx request.Context, options *internalversion.ListOptions) (*MachineSetList, error)
	GetMachineSet(ctx request.Context, id string, options *metav1.GetOptions) (*MachineSet, error)
	CreateMachineSet(ctx request.Context, id *MachineSet) (*MachineSet, error)
	UpdateMachineSet(ctx request.Context, id *MachineSet) (*MachineSet, error)
	DeleteMachineSet(ctx request.Context, id string) (bool, error)
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched types will panic.
func NewMachineSetRegistry(sp builders.StandardStorageProvider) MachineSetRegistry {
	return &storageMachineSet{sp}
}

// Implement Registry
// storage puts strong typing around storage calls
// +k8s:deepcopy-gen=false
type storageMachineSet struct {
	builders.StandardStorageProvider
}

func (s *storageMachineSet) ListMachineSets(ctx request.Context, options *internalversion.ListOptions) (*MachineSetList, error) {
	if options != nil && options.FieldSelector != nil && !options.FieldSelector.Empty() {
		return nil, fmt.Errorf("field selector not supported yet")
	}
	st := s.GetStandardStorage()
	obj, err := st.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineSetList), err
}

func (s *storageMachineSet) GetMachineSet(ctx request.Context, id string, options *metav1.GetOptions) (*MachineSet, error) {
	st := s.GetStandardStorage()
	obj, err := st.Get(ctx, id, options)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineSet), nil
}

func (s *storageMachineSet) CreateMachineSet(ctx request.Context, object *MachineSet) (*MachineSet, error) {
	st := s.GetStandardStorage()
	obj, err := st.Create(ctx, object, nil, true)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineSet), nil
}

func (s *storageMachineSet) UpdateMachineSet(ctx request.Context, object *MachineSet) (*MachineSet, error) {
	st := s.GetStandardStorage()
	obj, _, err := st.Update(ctx, object.Name, rest.DefaultUpdatedObjectInfo(object), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*MachineSet), nil
}

func (s *storageMachineSet) DeleteMachineSet(ctx request.Context, id string) (bool, error) {
	st := s.GetStandardStorage()
	_, sync, err := st.Delete(ctx, id, nil)
	return sync, err
}

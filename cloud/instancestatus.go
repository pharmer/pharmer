package cloud

import (
	"bytes"
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubernetes/pkg/apis/core"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Long term, we should retrieve the current status by asking k8s, gce etc. for all the needed info.
// For now, it is stored in the matching CRD under an annotation. This is similar to
// the spec and status concept where the machine CRD is the instance spec and the annotation is the instance status.

const InstanceStatusAnnotationKey = "instance-status"

type instanceStatus *clusterv1.Machine

type StatusManager struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewStatusManager(client client.Client, scheme *runtime.Scheme) *StatusManager {
	return &StatusManager{client, scheme}
}

func (sm *StatusManager) Initialize(machine *clusterv1.Machine) instanceStatus {
	return instanceStatus(machine)
}

// Get the status of the instance identified by the given machine
func (sm *StatusManager) InstanceStatus(machine *clusterv1.Machine) (instanceStatus, error) {
	if sm.client == nil {
		return nil, nil
	}
	namespace := machine.Namespace
	if machine.Namespace == "" {
		namespace = core.NamespaceDefault
	}
	currentMachine, err := util.GetMachineIfExists(sm.client, namespace, machine.ObjectMeta.Name)
	if err != nil {
		return nil, err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted (or does not exist yet ie. bootstrapping)
		return nil, fmt.Errorf("Machine %v not found", machine.Name)
	}
	return sm.machineInstanceStatus(currentMachine)
}

// Sets the status of the instance identified by the given machine to the given machine
func (sm *StatusManager) UpdateInstanceStatus(machine *clusterv1.Machine) error {
	if sm.client == nil {
		return nil
	}
	namespace := machine.Namespace
	if machine.Namespace == "" {
		namespace = core.NamespaceDefault
	}

	status := instanceStatus(machine)
	currentMachine, err := util.GetMachineIfExists(sm.client, namespace, machine.ObjectMeta.Name)
	if err != nil {
		return err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted.
		return fmt.Errorf("Machine has already been deleted. Cannot update current instance status for machine %v", machine.ObjectMeta.Name)
	}

	m, err := sm.SetMachineInstanceStatus(currentMachine, status)
	if err != nil {
		return err
	}

	return sm.client.Status().Update(context.Background(), m)
}

// Gets the state of the instance stored on the given machine CRD
func (sm *StatusManager) machineInstanceStatus(machine *clusterv1.Machine) (instanceStatus, error) {
	if machine.Annotations == nil {
		// No state
		return nil, nil
	}

	a := machine.ObjectMeta.Annotations[InstanceStatusAnnotationKey]
	if a == "" {
		// No state
		return nil, nil
	}

	serializer := json.NewSerializer(json.DefaultMetaFactory, sm.scheme, sm.scheme, false)
	var status clusterv1.Machine
	dvk := clusterv1.SchemeGroupVersion.WithKind("Machine")
	_, _, err := serializer.Decode([]byte(a), &dvk, &status)
	if err != nil {
		return nil, fmt.Errorf("decoding failure: %v", err)
	}

	return instanceStatus(&status), nil
}

// Applies the state of an instance onto a given machine CRD
func (sm *StatusManager) SetMachineInstanceStatus(machine *clusterv1.Machine, status instanceStatus) (*clusterv1.Machine, error) {
	// Avoid status within status within status ...
	if status.ObjectMeta.Annotations == nil {
		status.ObjectMeta.Annotations = make(map[string]string)
	}
	status.ObjectMeta.Annotations[InstanceStatusAnnotationKey] = ""

	serializer := json.NewSerializer(json.DefaultMetaFactory, sm.scheme, sm.scheme, false)
	b := []byte{}
	buff := bytes.NewBuffer(b)
	err := serializer.Encode((*clusterv1.Machine)(status), buff)
	if err != nil {
		return nil, fmt.Errorf("encoding failure: %v", err)
	}

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations[InstanceStatusAnnotationKey] = buff.String()
	return machine, nil
}

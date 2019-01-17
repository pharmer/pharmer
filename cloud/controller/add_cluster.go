package controller

import (
	"sigs.k8s.io/cluster-api-provider-gcp/pkg/cloud/google"
	"sigs.k8s.io/cluster-api/pkg/controller/cluster"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, func(m manager.Manager) error {
		actuator, err := google.NewClusterActuator(m, google.ClusterActuatorParams{})
		if err != nil {
			return err
		}
		return cluster.AddWithActuator(m, actuator)
	})
}

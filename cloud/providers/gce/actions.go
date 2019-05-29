package gce

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

const (
	network                = "Default Network"
	networkMessage         = "default network with ipv4 range 10.240.0.0/16"
	foundNetworkMessage    = "Found " + networkMessage
	notFoundNetworkMessage = "Not found, " + networkMessage + " will be created"
)

func addActs(acts []api.Action, action api.ActionType, resource, message string) []api.Action {
	return append(acts, api.Action{
		Action:   action,
		Resource: resource,
		Message:  message,
	})
}

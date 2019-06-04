package gce

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

const (
	network                = "Default Network"
	networkMessage         = "default network with ipv4 range 10.240.0.0/16"
	networkFoundMessage    = "Found " + networkMessage
	networkNotFoundMessage = "Not found, " + networkMessage + " will be created"

	firewall                = "Default Firewall Rule"
	firewallMessage         = "default-allow-internal, default-allow-ssh, https rules"
	firewallFoundMessage    = "Found " + firewallMessage
	firewallNotFoundMessage = "Not found, " + firewallMessage + " will be created"

	loadBalancer                = "Load Balancer"
	loadBalancerMessage         = loadBalancer
	loadBalancerFoundMessage    = "Found " + loadBalancerMessage
	loadBalancerNotFoundMessage = "Not found, " + loadBalancerMessage + " will be created"

	masterInstance                = "Master Instance"
	masterInstanceMessage         = masterInstance
	masterInstanceFoundMessage    = "Found " + masterInstanceMessage
	masterInstanceNotFoundMessage = "Not found, " + masterInstanceMessage + " will be created"
)

func addActs(acts []api.Action, action api.ActionType, resource, message string) []api.Action {
	return append(acts, api.Action{
		Action:   action,
		Resource: resource,
		Message:  message,
	})
}

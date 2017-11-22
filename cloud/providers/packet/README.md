## Running Kubernetes on Packet with Pharmer

[Packet](http://www.packet.net) is a bare metal cloud built for developers.

Pharmer uses the [Packet API](https://www.packet.net/developers/api/) for provisioning.

The following instructions use `pharmer` to create, delete, upgrade and scale up or down a Kubernetes cluster on Packet. Pharmer uses a Ubuntu 16.04 LTS image by default.

### Prerequisites

#### On Packet

* Create a Packet account
* Create a Packet project, or use an existing one
* Create a Packet API key for `pharmer` use

#### On your local system

* Install `go`
* Install `pharmer` 
* Install `kubeadm`

### Limitations

`pharmer` does not yet support the Packet spot market or reserved instances.

The Type 2A (arm64) servers are not yet tested.

### Create a cluster

Use the Packet project UUID and Packet API key for the `pharmer create credential` command.

```console
$ pharmer create credential p2 
```

### Delete a cluster
### Upgrade a cluster
### Scale a cluster up or down

## Example Commands

```console
$ pharmer create credential p2

$ pharmer create cluster packet \
	--v=5 \
	--provider=packet \
	--zone=ewr1 \
	--nodes=baremetal_0=0 \
	--credential-uid=p2 \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0' --kubeadm-version='1.8.0'

$ pharmer apply packet --v=3
```

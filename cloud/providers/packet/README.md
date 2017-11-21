## Running Kubernetes on Packet with Pharmer

[Packet](http://www.packet.net) is a bare metal cloud built for developers.

Pharmer uses the [Packet API](https://www.packet.net/developers/api/) for provisioning.

Before you start, you'll need a project UUID and API key for the `pharmer create credential` command.

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

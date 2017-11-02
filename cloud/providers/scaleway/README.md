
## Example Commands

```console
$ pharmer create credential sc

$ pharmer create cluster scaleway \
	--v=5 \
	--provider=scaleway \
	--zone=ams1 \
	--nodes=VC1S=0 \
	--credential-uid=sc \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0' --kubeadm-version='1.8.0'

$ pharmer apply scaleway --v=3
```

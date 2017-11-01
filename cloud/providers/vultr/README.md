
## Example Commands

```console
$ pharmer create credential v2

$ pharmer create cluster vultr \
	--v=5 \
	--provider=vultr \
	--zone=6 \
	--nodes=94=0 \
	--credential-uid=v2 \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0*' --kubeadm-version='1.8.0*'

$ pharmer apply vultr --v=3
```

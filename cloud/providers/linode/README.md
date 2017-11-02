
## Example Commands

```console
$ pharmer create credential l2

$ pharmer create cluster linode \
	--v=5 \
	--provider=linode \
	--zone=3 \
	--nodes=1=0 \
	--credential-uid=l2 \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0' --kubeadm-version='1.8.0'

$ pharmer apply linode --v=3
```

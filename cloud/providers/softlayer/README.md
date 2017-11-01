
## Example Commands

```console
$ pharmer create credential sl

$ pharmer create cluster softlayer \
	--v=5 \
	--provider=softlayer \
	--zone=dal05 \
	--nodes=2c2m=0 \
	--credential-uid=sl \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0*' --kubeadm-version='1.8.0*'

$ pharmer apply softlayer --v=3
```

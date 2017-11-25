
## Example Commands

```console
$ pharmer create credential aws

$ pharmer create cluster c1 \
	--v=5 \
	--provider=lightsail \
	--zone=us-west-2a \
	--nodes=small_1_0=1 \
	--credential-uid=aws \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0' --kubeadm-version='1.8.0'

$ pharmer apply c1 --v=3
```

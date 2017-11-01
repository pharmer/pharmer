
## Example Commands

```console
$ pharmer create credential d2

$ pharmer create cluster c1 \
	--v=5 \
	--provider=digitalocean \
	--zone=nyc3 \
	--nodes=2gb=0 \
	--credential-uid=d2 \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0*' --kubeadm-version='1.8.0*'

$ pharmer apply c1 --v=3
```


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
	--binary-version='1.8.0*'

$ pharmer apply packet --v=3
```

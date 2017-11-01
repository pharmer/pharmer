
## Example Commands

```console
$ pharmer create credential sl

$ pharmer create cluster softlayer \
	--v=5 \
	--provider=bluemix \
	--zone=dal \
	--nodes=2c2m=0 \
	--credential-uid=sl \
	--kubernetes-version=1.8.0 \
	--binary-version='1.8.0*'

$ pharmer apply softlayer --v=3
```

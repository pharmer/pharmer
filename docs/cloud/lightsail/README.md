# Running Kubernetes on [Lightsail](https://lightsail.aws.amazon.com/)

Following example will use `pharmer ` to create a Kubernetes cluster with 2 worker node instances and a master instance (i,e, 3 instances in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.
```console
$ go get github.com/appscode/pharmer
```

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage
provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.

### Credential importing




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

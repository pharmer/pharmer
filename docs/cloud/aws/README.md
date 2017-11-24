# Running Kubernetes on [AWS](https://aws.amazon.com/)

Following example will use `pharmer ` to create a Kubernetes cluster with 2 worker node servers and a master server (i,e, 3 servers in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.
```console
$ go get github.com/appscode/pharmer
```

### AWS IAM Permissions

   

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage
provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.



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
	--kubelet-version='1.8.0' --kubeadm-version='1.8.0'

$ pharmer apply c1 --v=3
```

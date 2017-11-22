# Pod Networking

Available network provider support in `pharmer`
* [calico](https://kubernetes.io/docs/concepts/cluster-administration/networking/#project-calico)
* [flannel](https://kubernetes.io/docs/concepts/cluster-administration/networking/#flannel)
* [weavenet](https://kubernetes.io/docs/concepts/cluster-administration/networking/#weave-net-from-weaveworks)

`pharmer` uses `calico` as a default network provider. If you want to change the default network provider, you can do that with the help of following flag.
`--networking=<provider-name>`
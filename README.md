# pharmer
Kubernetes Cluster Manager for [Kubeadm](https://github.com/kubernetes/kubeadm). Think `kops using kubadm`!

This project is spread over 4 repositories:

- [appscode/pre-k](https://github.com/appscode/pre-k): Contains [a set of handy commands](https://github.com/appscode/pre-k/blob/master/docs/reference/pre-k.md) that you run before `kubeadm init`.

- [appscode/pharmer](https://github.com/appscode/pharmer): A [Kops](https://github.com/kubernetes/kops) [like](https://github.com/appscode/pharmer/blob/master/docs/reference/pharmer.md) cluster manager using Kubeadm. Supported cloud providers:
  - aws
  - azure
  - digitalocean
  - gce
  - linode
  - packet
  - scaleway
  - softlayer
  - vultr

- [appscode/pharm-controller-manager](https://github.com/appscode/pharm-controller-manager): Implements Cloud Controller manager for following cloud providers:
  - linode
  - packet
  - scaleway
  - softlayer
  - vultr

- [appscode/swanc](https://github.com/appscode/swanc): StrongSwan based VPN Controller for Kubernetes

## User Guide
 - [Create & manage a Kubernetes cluster in DigitalOcean](https://github.com/appscode/pharmer/blob/master/cloud/providers/digitalocean/README.md)
 - [Create & manage a Kubernetes cluster in Vultr](https://github.com/appscode/pharmer/blob/master/cloud/providers/vultr/README.md)

## Support
If you have any questions, [file an issue](https://github.com/appscode/pharmer/issues/new) or talk to us on the [Kubernetes Slack team](http://slack.kubernetes.io/) channel `#pharmer`.

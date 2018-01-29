[![Go Report Card](https://goreportcard.com/badge/github.com/pharmer/pharmer)](https://goreportcard.com/report/github.com/pharmer/pharmer)

# pharmer
Kubernetes Cluster Manager for [Kubeadm](https://github.com/kubernetes/kubeadm). Think `kops using kubeadm`!

This project is spread over 5 repositories:

- [pharmer/pre-k](https://github.com/pharmer/pre-k): Contains [a set of handy commands](https://github.com/pharmer/pre-k/blob/master/docs/reference/pre-k.md) that you run before `kubeadm init`.

- [pharmer/pharmer](https://github.com/pharmer/pharmer): A [Kops](https://github.com/kubernetes/kops) [like](https://github.com/pharmer/pharmer/blob/master/docs/reference/pharmer.md) cluster manager using `Kubeadm`. Supported cloud providers:
  - [Amazon Web Services](https://aws.amazon.com/)
  - [Amazon Lightsail](https://amazonlightsail.com/)
  - [DigitalOcean](https://www.digitalocean.com/)
  - [Google Cloud](https://cloud.google.com/compute/)
  - [Linode](https://www.linode.com/)
  - [Microsoft Azure](https://azure.microsoft.com/en-us/)
  - [Packet](https://www.packet.net/)
  - [Scaleway](https://www.scaleway.com/)
  - [Softlayer](http://www.softlayer.com/)
  - [Vultr](https://www.vultr.com/)

- [pharmer/cloud-controller-manager](https://github.com/pharmer/cloud-controller-manager): Implements Cloud Controller manager for following cloud providers:
  - [Amazon Lightsail](https://amazonlightsail.com/)
  - [Linode](https://www.linode.com/)
  - [Packet](https://www.packet.net/)
  - [Scaleway](https://www.scaleway.com/)
  - [Softlayer](http://www.softlayer.com/)
  - [Vultr](https://www.vultr.com/)

- [pharmer/flexvolumes](https://github.com/pharmer/flexvolumes): Implements Flex Volume drivers for following cloud providers:
  - [DigitalOcean](https://www.digitalocean.com/)
  - [Linode](https://www.linode.com/)
  - [Packet](https://www.packet.net/)

- [pharmer/swanc](https://github.com/pharmer/swanc): StrongSwan based VPN Controller for Kubernetes

## User Guide
 - [Create & manage a Kubernetes cluster in AWS EC2](/docs/cloud/aws/README.md)
 - [Create & manage a Kubernetes cluster in Amazon Lightsail](/docs/cloud/lightsail/README.md)
 - [Create & manage a Kubernetes cluster in Google Cloud](/docs/cloud/gce/README.md)
 - [Create & manage a Kubernetes cluster in Microsoft Azure](/docs/cloud/azure/README.md)
 - [Create & manage a Kubernetes cluster in DigitalOcean](/docs/cloud/digitalocean/README.md)
 - [Create & manage a Kubernetes cluster in Linode](/docs/cloud/linode/README.md)
 - [Create & manage a Kubernetes cluster in Packet](/docs/cloud/packet/README.md)
 - [Create & manage a Kubernetes cluster in Scaleway](/docs/cloud/scaleway/README.md)
 - [Create & manage a Kubernetes cluster in Vultr](/docs/cloud/vultr/README.md)

## Supported Versions
Kubernetes 1.8 & 1.9

## Contribution guidelines
Want to help improve Pharmer? Please start [here](/CONTRIBUTING.md).

## Support
If you have any questions, [file an issue](https://github.com/pharmer/pharmer/issues/new) or talk to us on the [Kubernetes Slack team](http://slack.kubernetes.io/) channel `#pharmer`.

---

**`pharmer` binary collects anonymous usage statistics to help us learn how the software is being used and how we can improve it.
To disable stats collection, run the operator with the flag** `--analytics=false`.

---

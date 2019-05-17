[![Go Report Card](https://goreportcard.com/badge/github.com/pharmer/pharmer)](https://goreportcard.com/report/github.com/pharmer/pharmer)
[![Build Status](https://travis-ci.org/pharmer/pharmer.svg?branch=master)](https://travis-ci.org/pharmer/pharmer)
[![codecov](https://codecov.io/gh/pharmer/pharmer/branch/master/graph/badge.svg)](https://codecov.io/gh/pharmer/pharmer)
[![Slack](http://slack.kubernetes.io/badge.svg)](http://slack.kubernetes.io/#pharmer)
[![Twitter](https://img.shields.io/twitter/follow/appscodehq.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=AppsCodeHQ)

# pharmer
<img src="https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/certified-kubernetes/1.13/color/certified-kubernetes-1.13-color.png" align="right" width="200px">Kubernetes Cluster Manager for [Kubeadm](https://github.com/kubernetes/kubeadm). Think `kops using kubeadm`!

This project is spread over 5 repositories:

- [pharmer/pre-k](https://github.com/pharmer/pre-k): Contains [a set of handy commands](https://github.com/pharmer/pre-k/blob/master/docs/reference/pre-k.md) that you run before `kubeadm init`.

- [pharmer/pharmer](https://github.com/pharmer/pharmer): A [Kops](https://github.com/kubernetes/kops) [like](https://github.com/pharmer/pharmer/blob/master/docs/reference/pharmer.md) cluster manager using `Kubeadm`. Supported cloud providers:
  - [Amazon Web Services](https://aws.amazon.com/)
  - [Amazon EKS](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)
  - [DigitalOcean](https://www.digitalocean.com/)
  - [Google Cloud](https://cloud.google.com/compute/)
  - [Google Kubernetes Engine GKE](https://cloud.google.com/kubernetes-engine/)
  - [Linode](https://www.linode.com/)
  - [Microsoft Azure](https://azure.microsoft.com/en-us/)
  - [Azure Kubernetes Servic AKS](https://docs.microsoft.com/en-us/azure/aks/)
  - [Packet](https://www.packet.net/)
  <!-- - [Scaleway](https://www.scaleway.com/)
  - [Softlayer](http://www.softlayer.com/)
  - [Vultr](https://www.vultr.com/) -->

- [pharmer/cloud-controller-manager](https://github.com/pharmer/cloud-controller-manager): Implements Cloud Controller manager for following cloud providers:
  - [Linode](https://www.linode.com/)
  - [Packet](https://www.packet.net/)
  - [Digitalocean](https://digitalocean.com)
  <!-- - [Scaleway](https://www.scaleway.com/) -->
  <!-- - [Softlayer](http://www.softlayer.com/) -->
  <!-- - [Vultr](https://www.vultr.com/) -->

- [pharmer/flexvolumes](https://github.com/pharmer/flexvolumes): Implements Flex Volume drivers for following cloud providers:
  - [DigitalOcean](https://www.digitalocean.com/)
  - [Linode](https://www.linode.com/)
  - [Packet](https://www.packet.net/)

- [pharmer/swanc](https://github.com/pharmer/swanc): StrongSwan based VPN Controller for Kubernetes

## User Guide
 - [Create & manage a Kubernetes cluster in AWS EC2](/docs/cloud/aws/README.md)
 - [Create & manage a Kubernetes cluster in Amazon EKS](/docs/cloud/eks/README.md)
 - [Create & manage a Kubernetes cluster in Google Cloud](/docs/cloud/gce/README.md)
 - [Create & manage a Kubernetes cluster in Google Kubernetes Engine](/docs/cloud/gke/README.md)
 - [Create & manage a Kubernetes cluster in Microsoft Azure](/docs/cloud/azure/README.md)
 - [Create & manage a Kubernetes cluster in Azure Kubernetes Servic](/docs/cloud/aks/README.md)
 - [Create & manage a Kubernetes cluster in DigitalOcean](/docs/cloud/digitalocean/README.md)
 - [Create & manage a Kubernetes cluster in Linode](/docs/cloud/linode/README.md)
 - [Create & manage a Kubernetes cluster in Packet](/docs/cloud/packet/README.md)
 <!-- - [Create & manage a Kubernetes cluster in Scaleway](/docs/cloud/scaleway/README.md)
 - [Create & manage a Kubernetes cluster in Vultr](/docs/cloud/vultr/README.md) -->

## Supported Versions Matrix

| pharmer version | k8s 1.9.x | k8s 1.10.x | k8s 1.11.x | k8s 1.12.x | k8s 1.13.x | k8s 1.14.x
|-----------------|-----------|------------|------------|------------|---------|---------------
| 0.3.0           | &#10007;  | &#10007;   | &#10007;   |&#10007;    | &#10003;| &#10003;
| 0.2.0           | &#10007;  | &#10007;   | &#10007;   | &#10003;   | &#10003;| &#10007;
| 0.1.1           | &#10007;  | &#10007;   | &#10003;   | &#10003;   | &#10007;| &#10007;
| 0.1.0-rc.5      | &#10007;  | &#10003;   | &#10003;   | &#10007;   | &#10007;| &#10007;
| 0.1.0-rc.4      | &#10003;  | &#10003;   | &#10007;   | &#10007;   | &#10007;| &#10007;

## Contribution guidelines
Want to help improve Pharmer? Please start [here](/CONTRIBUTING.md).

---

**Pharmer binaries collects anonymous usage statistics to help us learn how the software is being used and how we can improve it. To disable stats collection, run the operator with the flag** `--analytics=false`.

---

## Support
We use Slack for public discussions. To chit chat with us or the rest of the community, join us in the [Kubernetes Slack team](https://kubernetes.slack.com/messages/C81LSKMPE/details/) channel `#pharmer`. To sign up, use our [Slack inviter](http://slack.kubernetes.io/).

To receive product announcements, please join our [mailing list](https://groups.google.com/forum/#!forum/pharmer) or follow us on [Twitter](https://twitter.com/AppsCodeHQ). Our mailing list is also used to share design docs shared via Google docs.

If you have found a bug with Pharmer or want to request for new features, please [file an issue](https://github.com/pharmer/pharmer/issues/new).

---
title: Networking | Pharmer
description: Networking of Pharmer
menu:
  product_pharmer_0.3.1:
    identifier: networking-pharmer
    name: Networking
    parent: getting-started
    weight: 30
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: getting-started
url: /products/pharmer/0.3.1/getting-started/networking/
aliases:
  - /products/pharmer/0.3.1/networking/
---

# Pod Networking

Available network provider support in `pharmer`
* [calico](https://kubernetes.io/docs/concepts/cluster-administration/networking/#project-calico)
* [flannel](https://kubernetes.io/docs/concepts/cluster-administration/networking/#flannel)
* [weavenet](https://kubernetes.io/docs/concepts/cluster-administration/networking/#weave-net-from-weaveworks)

`pharmer` uses `calico` as a default network provider. If you want to change the default network provider, you can do that with the help of following flag.
`--networking=<provider-name>`
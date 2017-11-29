---
title: GCE Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: gce-overview
    name: Overview
    parent: gce
    weight: 10
product_name: pharmer
left_menu: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/gce/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/gce/README/
---

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

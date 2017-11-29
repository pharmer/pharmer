---
title: Azure Overview
menu:
  product_pharmer_0.2.0:
    identifier: azure-overview
    name: Overview
    parent: azure
    weight: 10
product_name: pharmer
left_menu: product_pharmer_0.2.0
section_menu_id: cloud
url: /products/pharmer/0.2.0/cloud/azure/
aliases:
  - /products/pharmer/0.2.0/cloud/azure/README/
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

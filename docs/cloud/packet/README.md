---
title: Packet Overview
menu:
  product_pharmer_0.2.0:
    identifier: packet-overview
    name: Overview
    parent: packet
    weight: 10
product_name: pharmer
left_menu: product_pharmer_0.2.0
section_menu_id: cloud
url: /products/pharmer/0.2.0/cloud/packet/
aliases:
  - /products/pharmer/0.2.0/cloud/packet/README/
---

## Example Commands

```console
$ pharmer create credential p2

$ pharmer create cluster packet \
	--v=5 \
	--provider=packet \
	--zone=ewr1 \
	--nodes=baremetal_0=1 \
	--credential-uid=p2 \
	--kubernetes-version=1.8.0 \
	--kubelet-version='1.8.0' --kubeadm-version='1.8.0'

$ pharmer apply packet --v=3
```

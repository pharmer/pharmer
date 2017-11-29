---
title: Linode Overview
menu:
  product_pharmer_0.2.0:
    identifier: linode-overview
    name: Overview
    weight: 10
product_name: pharmer
left_menu: product_pharmer_0.2.0
section_menu_id: cloud
url: /products/pharmer/0.2.0/cloud/linode/
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

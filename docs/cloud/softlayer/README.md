---
title: SoftLayer Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: softlayer-overview
    name: Overview
    parent: soft-layer
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/softlayer/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/softlayer/README/
---

## Example Commands

```console
$ pharmer create credential sl

$ pharmer create cluster softlayer \
	--v=5 \
	--provider=softlayer \
	--zone=dal05 \
	--nodes=2c2m=0 \
	--credential-uid=sl \
	--kubernetes-version=1.8.0

$ pharmer apply softlayer --v=3
```

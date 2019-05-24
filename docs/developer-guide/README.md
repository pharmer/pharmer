---
title: Overview | Developer Guide
description: Developer Guide Overview
menu:
  product_pharmer_0.3.0:
    identifier: developer-guide-readme
    name: Overview
    parent: developer-guide
    weight: 15
product_name: pharmer
menu_name: product_pharmer_0.3.0
section_menu_id: developer-guide
url: /products/pharmer/0.3.0/developer-guide/
aliases:
  - /products/pharmer/0.3.0/developer-guide/README/
---

## Development Guide
This document is intended to be the canonical source of truth for things like supported toolchain versions for building Pack.
If you find a requirement that this doc does not capture, please submit an issue on github.

This document is intended to be relative to the branch in which it is found. It is guaranteed that requirements will change over time
for the development branch, but release branches of Pharmer should not change.

### Build Pack
Some of the Pharmer development helper scripts rely on a fairly up-to-date GNU tools environment, so most recent Linux distros should
work just fine out-of-the-box.

#### Setup GO
Pharmer is written in Google's GO programming language. Currently, Pharmer is developed and tested on **go 1.8.3**. If you haven't set up a GO
development environment, please follow [these instructions](https://golang.org/doc/code.html) to install GO.

#### Download Source

```console
$ go get -u github.com/pharmer/pharmer
$ cd $(go env GOPATH)/src/github.com/pharmer/pharmer
```

#### Install Dev tools
To install various dev tools for Pack, run the following command:

```console
# setting up dependencies for compiling pharmer...
$ ./hack/builddeps.sh
```

#### Build Binary
```
$ ./hack/make.py
$ pharmer version
```

#### Dependency management
Pharmer uses [Glide](https://github.com/Masterminds/glide) to manage dependencies. Dependencies are already checked in the `vendor` folder.
If you want to update/add dependencies, run:
```console
$ glide slow
```

#### Generate CLI Reference Docs
```console
$ ./hack/gendocs/make.sh
```

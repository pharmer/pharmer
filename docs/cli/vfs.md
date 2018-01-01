---
title: Virtual File System
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: vfs
    name: VFS
    parent: cli
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cli
aliases:
  - /products/pharmer/0.1.0-alpha.1/cli/
---

## Virtual file system (Vfs)

Stores data on File (local/remote) using  [stow](https://github.com/appscode/stow)

### Introduction

```console
$  pharmer config -h
  Pharmer configuration

  Usage:
    pharmer config [flags]
    pharmer config [command]

  Examples:
  pharmer config view

  Available Commands:
    get-contexts List available contexts
    view         Print Pharmer config

  Flags:
    -h, --help   help for config

  Global Flags:
        --alsologtostderr                  log to standard error as well as files
        --analytics                        Send analytical events to Google Guard (default true)
        --config-file string               Path to Pharmer config file
        --env string                       Environment used to enable debugging (default "dev")
        --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
        --log_dir string                   If non-empty, write log files in this directory
        --logtostderr                      log to standard error instead of files (default true)
        --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
    -v, --v Level                          log level for V logs
        --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### Configuration

The configuration details of the storage provider are specified on `~/.pharmer/config.d/default` file.

```yaml
context: default
kind: PharmerConfig
store:
  local:
    path: /home/sanjid/.pharmer/store.d
```
Here store type is `local`, so in `path` a local directory is used to locate where the cluster and credential resources will be stored.

You can also use Amazon's `s3`, `gcs` to use google cloud storage, `azure` or `swift` for storage purpose.
For using `s3` you have to modify the configuration file with following field
```yaml
  s3:
    endpoint: <aws endpoint>
    bucket: <bucket name>
    prefix: <storage prefix>
```
To use `gcs` modify with
```yaml
context: default
kind: PharmerConfig
credentials:
- metadata:
    creationTimestamp: 2018-01-01T09:01:19Z
    name: gce
  spec:
    data:
      projectID: <project-id>
      serviceAccount: |-
        {
          "type": "service_account",
          "project_id": <project-id>,
          "private_key_id": "private key id",
          "private_key": "private key",
          "client_email": "email",
          "client_id": "client id",
          "auth_uri": "https://accounts.google.com/o/oauth2/auth",
          "token_uri": "https://accounts.google.com/o/oauth2/token",
          "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
          "client_x509_cert_url": "url"
        }
    provider: GoogleCloud
store:
  credentialName: gce
  gcs:
    bucket: pharmer
```
For `azure` and `swift` you need to add `container` field along with `prefix` field.

### Storage structure

The directory tree of the local storage provider will be look like:

```console
~/.pharmer/
      |--config.d/
      |      |
      |      |__ default
      |
      |__ store.d/
             |
             |-- clusters/
             |
             |__ credentials/

```

Here,
 - `config.d/default`: is the storage configuration file
 - `store.d/cluster`: stores cluster resources. There is a file with `<cluster-name>.json`, a `nodegroup` directory which contains
 nodegroup files, a `pki` directory having cluster certificates and an `ssh` directory which stores public and private ssh key for the cluster.
 - `store.d/credentials`: stores credential resources

You can view the config using
```yaml
$ pharmer config view

pharmer config view
context: default
kind: PharmerConfig
store:
  local:
    path: /home/sanjid/.pharmer/store.d

```

To list all available contexts run:
```console
$ pharmer config get-contexts
NAME	Store
default	Local

```

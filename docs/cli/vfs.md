---
title: Virtual File System
menu:
  product_pharmer_0.3.1:
    identifier: vfs
    name: VFS
    parent: cli
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: cli
aliases:
  - /products/pharmer/0.3.1/cli/
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
  set-context  Create  config object
  view         Print Pharmer config

Flags:
  -h, --help   help for config

Global Flags:
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Guard (default true)
      --config-file string               Path to Pharmer config file
      --env string                       Environment used to enable debugging (default "prod")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging

Use "pharmer config [command] --help" for more information about a command.
```

### Configuration

The configuration details of the storage provider are specified on `~/.pharmer/config.d/default` file. Default storage provider is `local`.

```yaml
context: default
kind: PharmerConfig
store:
  local:
    path: /home/sanjid/.pharmer/store.d
```
Here store type is `local`, so in `path` a local directory is used to locate where the cluster and credential resources will be stored.

You can also use Amazon's `s3`, `gcs` to use google cloud storage, `azure` or `swift` for storage purpose.

You can use following command to crate a storage provider confiuration.

```console
# AWS S3:
pharmer config set-context --provider=s3 --s3.access_key_id=<key_id> --s3.secret_key=<secret_key> --s3.endpoint=<endpoint> --s3.bucket=<bucket_name> --prefix=<prefix>

# GCS:
pharmer config set-context --provider=google --google.json_key_path=<path_sa_file> --google.project_id=<my_project> --google.bucket=<bucket_name> --prefix=<prefix>

# Microsoft Azure ARM Storage:
pharmer config set-context --provider=azure --azure.account=<storage_ac> --azure.key=<key> --azure.container=<container_name> --prefix=<prefix>

# Local Storage:
pharmer config set-context --provider=local --local.path=<local_path>

# Swift:
pharmer config set-context --provider=swift --swift.key=<key> --swift.tenant_auth_url=<tenant_auth_url> --swift.tenant_name=<tenant_name> --swift.username=<username>
--swift.domain=<domain> --swift.region=<region> --swift.tenant_id=<tenant_id> --swift.tenant_domain=<tenant_domain> --swift.storage_url=<storage_url>
--swift.auth_token=<auth_token> --swift.container=<container_name> --prefix=<prefix>

```


If you using `s3`, the configuration file contains following field
```yaml
  s3:
    endpoint: <aws endpoint>
    bucket: <bucket name>
    prefix: <storage prefix>
```

For `gcs`
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

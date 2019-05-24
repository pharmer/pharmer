---
title: Postgres Database XORM
menu:
  product_pharmer_0.3.1:
    identifier: xorm
    name: XORM
    parent: cli
    weight: 15
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: cli
---

## Postgres Database (xorm)

Stores data in PostgresSQL using [xorm](https://github.com/go-xorm/xorm)


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

The configuration details of the storage provider are specified on `~/.pharmer/config.d/default` file.

To create the storage you can run the following command

```console
$ pharmer config set-context --pg.db-name=<dbName> --pg.host=<host> --pg.port=5432 --pg.user=<user> --pg.password=<password>

```

The configuration file look like:

```yaml
context: default
kind: PharmerConfig
store:
  postgres:
    database: <database name>
    user: <database user>
    password: <password>
    host: 127.0.0.1
    port: 5432
```

### Storage structure

The table of the xorm storage provider will be look like:

**cluster**

|id|kind|apiVersion|name|uid|resourceVersion|generation|labels|data|creationTimestamp|dateModified|deletionTimestamp|
|--|----|----------|----|---|---------------|----------|------|----|-----------------|------------|-----------------|

**credential**

|id|kind|apiVersion|name|uid|resourceVersion|generation|labels|data|creationTimestamp|dateModified|deletionTimestamp|
|--|----|----------|----|---|---------------|----------|------|----|-----------------|------------|-----------------|

**nodegroup**

|id|kind|apiVersion|name|clusterName|uid|resourceVersion|generation|labels|data|creationTimestamp|dateModified|deletionTimestamp|
|--|----|----------|----|-----------|---|---------------|----------|------|----|-----------------|------------|-----------------|

**certificate**

|id|name|clusterName|uid|cert|key|creationTimestamp|dateModified|deletionTimestamp|
|--|----|-----------|---|----|---|-----------------|------------|-----------------|

**certificate**

|id|name|clusterName|uid|publicKey|privateKey|creationTimestamp|dateModified|deletionTimestamp|
|--|----|-----------|---|---------|----------|-----------------|------------|-----------------|


You can view the config using
```yaml
$ pharmer config view

pharmer config view
context: default
kind: PharmerConfig
store:
  store:
    postgres:
      database: postgres
      user: postgres
      password: postgres
      host: 127.0.0.1
      port: 5432

```

To list all available contexts run:
```console
$ pharmer config get-contexts
NAME	Store
default	Postgres

```
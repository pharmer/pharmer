## Postgres Database (xorm)

Stores data in PostgresSQL using [xorm](https://github.com/go-xorm/xorm)
 

### Introduction

```bash
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

```bash
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
```bash
$ pharmer config get-contexts
NAME	Store
default	Local

```
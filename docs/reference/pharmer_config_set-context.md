---
title: Pharmer Config Set-Context
menu:
  product_pharmer_0.3.0:
    identifier: pharmer-config-set-context
    name: Pharmer Config Set-Context
    parent: reference
product_name: pharmer
menu_name: product_pharmer_0.3.0
section_menu_id: reference
---
## pharmer config set-context

Create  config object

### Synopsis

Create  config object

```
pharmer config set-context [flags]
```

### Examples

```
pharmer config set-context
```

### Options

```
      --azure.account string           Azure config account
      --azure.container string         Azure container name
      --azure.key string               Azure config key
      --google.bucket string           GCS config scopes
      --google.json_key_path string    GCS config json key path
      --google.project_id string       GCS config project id
  -h, --help                           help for set-context
      --local.path string              Local config key path
      --pg.db-name string              Postgres databases name
      --pg.host string                 Postgres host address
      --pg.password string             Postgres user password
      --pg.port int                    Postgres port number (default 5432)
      --pg.user string                 Postgres database user
      --provider string                Cloud storage provider
      --s3.access_key_id string        S3 config access key id
      --s3.bucket string               S3 store bucket
      --s3.endpoint string             S3 storage endpoint
      --s3.secret_key string           S3 config secret key
      --swift.auth_token string        Swift AuthToken
      --swift.container string         Swift container name
      --swift.domain string            Swift domain
      --swift.key string               Swift config key
      --swift.region string            Swift region
      --swift.storage_url string       Swift StorageURL
      --swift.tenant_auth_url string   Swift teanant auth url
      --swift.tenant_domain string     Swift TenantDomain
      --swift.tenant_id string         Swift TenantId
      --swift.tenant_name string       Swift tenant name
      --swift.username string          Swift username
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Guard (default true)
      --config-file string               Path to Pharmer config file
      --env string                       Environment used to enable debugging (default "prod")
      --kubeconfig string                Paths to a kubeconfig. Only required if out-of-cluster.
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --master --kubeconfig              (Deprecated: switch to --kubeconfig) The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [pharmer config](/docs/reference/pharmer_config.md)	 - Pharmer configuration


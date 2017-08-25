## pharmer credential

Manage cloud provider credentials

### Synopsis


Manage cloud provider credentials

```
pharmer credential [flags]
```

### Options

```
  -h, --help   help for credential
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --config-file string               Path to Pharmer config file
      --env string                       Environment used to enable debugging (default "dev")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [pharmer](pharmer.md)	 - Pharmer by Appscode - Manages farms
* [pharmer credential create](pharmer_credential_create.md)	 - Create credential for cloud providers (example, AWS, Google Cloud Platform)
* [pharmer credential delete](pharmer_credential_delete.md)	 - Delete a cloud credential
* [pharmer credential list](pharmer_credential_list.md)	 - List cloud credentials
* [pharmer credential update](pharmer_credential_update.md)	 - Update an existing cloud credential


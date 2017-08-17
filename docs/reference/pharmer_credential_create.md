## pharmer credential create

Create credential for cloud providers (example, AWS, Google Cloud Platform)

### Synopsis


Create credential for cloud providers (example, AWS, Google Cloud Platform)

```
pharmer credential create [flags]
```

### Examples

```
appctl credential create -p aws mycred
appctl credential create -p azure mycred
appctl credential create -p gce mycred
```

### Options

```
  -h, --help              help for create
  -p, --provider string   Cloud provider name (e.g., aws, gce, azure)
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [pharmer credential](pharmer_credential.md)	 - Manage cloud provider credentials


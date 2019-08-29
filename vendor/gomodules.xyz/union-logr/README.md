# union-logr
A [logr](https://github.com/go-logr/logr) implementation that aggregates multiple loggers.

Usage
---

### Code Example
```go
package main

import (
	"flag"
	"fmt"

	"github.com/go-logr/glogr"
	ulogr "gomodules.xyz/union-logr"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
)

func main(){
	flag.Set("logtostderr", "true")
	flag.Parse()
	logG := glogr.New().WithName("glog")

	klog.InitFlags(flag.NewFlagSet("klog", flag.ExitOnError))
	logK := klogr.New().WithName("klog")

	ulog := ulogr.NewLogger(logG, logK).WithName("ulog").WithValues("logr", "union-logr")
	ulog.V(0).Info("Example", "Key", "Value")
}

```
### Description

For using [union-logr](https://github.com/gomodules/union-logr), you just need to do the followings:

- Define some logger (i.e.: `glogr`, `klogr` etc.)
- Pass those logger to `ulogr.NewUnionLogger` and use it like you are using a single logger.

Thus, you can use multiple loggers at a time using a single [union-logr](https://github.com/gomodules/union-logr). 


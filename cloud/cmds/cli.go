package cmds

import (
	"k8s.io/client-go/util/homedir"
)

func KubeConfigPath() string {
	return homedir.HomeDir() + "/.kube/config"
}

package cloud

import (
	"fmt"
	"strings"
	"testing"
)

func TestUpgradeScript(t *testing.T) {
	// ref: https://stackoverflow.com/a/2831449/244009
	script := []string{
		`echo "#!/bin/bash" > /tmp/pharmer.sh`,
		`echo "set -xeou pipefail" >> /tmp/pharmer.sh`,
		`echo "" >> /tmp/pharmer.sh`,
		`echo "apt-get update" >> /tmp/pharmer.sh`,
		// `echo "apt-get upgrade -y kubelet kubectl" >> /tmp/pharmer.sh`,
		// `echo "systemctl restart kubelet" >> /tmp/pharmer.sh`,
		"chmod +x /tmp/pharmer.sh",
		"nohup /tmp/pharmer.sh > /var/log/pharmer.log 2>&1 &",
	}
	cmd := fmt.Sprintf("sh -c '%s'", strings.Join(script, "; "))

	fmt.Println(cmd)
}

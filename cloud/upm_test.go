package cloud

import (
	"fmt"
	"strings"
	"testing"
)

func TestUpgradeScript(t *testing.T) {
	// ref: https://stackoverflow.com/a/2831449/244009
	script := []string{
		`echo "#!/bin/bash" > /usr/bin/pharmer.sh`,
		`echo "set -xeou pipefail" >> /usr/bin/pharmer.sh`,
		`echo "" >> /usr/bin/pharmer.sh`,
		`echo "apt-get update" >> /usr/bin/pharmer.sh`,
		// `echo "apt-get upgrade -y kubelet kubectl" >> /usr/bin/pharmer.sh`,
		// `echo "systemctl restart kubelet" >> /usr/bin/pharmer.sh`,
		"chmod +x /usr/bin/pharmer.sh",
		"nohup /usr/bin/pharmer.sh > /var/log/pharmer.log 2>&1 &",
	}
	cmd := fmt.Sprintf("sh -c '%s'", strings.Join(script, "; "))

	fmt.Println(cmd)
}

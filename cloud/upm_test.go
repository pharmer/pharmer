package cloud

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	version "gomodules.xyz/version"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("upgrade script", func() {

		script := []string{
			`echo "#!/bin/bash" > /usr/bin/pharmer.sh`,
			`echo "set -xeou pipefail" >> /usr/bin/pharmer.sh`,
			`echo "" >> /usr/bin/pharmer.sh`,
			`echo "apt-get update" >> /usr/bin/pharmer.sh`,

			"chmod +x /usr/bin/pharmer.sh",
			"nohup /usr/bin/pharmer.sh > /var/log/pharmer.log 2>&1 &",
		}
		cmd := fmt.Sprintf("sh -c '%s'", strings.Join(script, "; "))

		fmt.Println(cmd)
	})
	It("version", func() {

		v1, _ := version.NewVersion("1.11.3")
		v2, _ := version.NewVersion("1.11.0")
		fmt.Println(v1.Compare(v2))
	})
})

package linode

import (
	"bytes"
	"context"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
)

var (
	customTemplate = `
{{ define "prepare-host" }}
# http://ask.xmodulo.com/disable-ipv6-linux.html
/bin/cat >>/etc/sysctl.conf <<EOF
# to disable IPv6 on all interfaces system wide
net.ipv6.conf.all.disable_ipv6 = 1

# to disable IPv6 on a specific interface (e.g., eth0, lo)
net.ipv6.conf.lo.disable_ipv6 = 1
net.ipv6.conf.eth0.disable_ipv6 = 1
EOF
/sbin/sysctl -p /etc/sysctl.conf
/bin/sed -i 's/^#AddressFamily any/AddressFamily inet/' /etc/ssh/sshd_config

export DEBIAN_FRONTEND=noninteractive
export DEBCONF_NONINTERACTIVE_SEEN=true
/usr/bin/apt-get update
/usr/bin/apt-get install -y --no-install-recommends --force-yes linux-image-amd64 grub2

{{ end }}
`
)

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func renderStartupScript(ctx context.Context, cluster *api.Cluster, role string) (string, error) {
	tpl, err := StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	tpl, err = tpl.Parse(customTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, role, GetTemplateData(ctx, cluster, "", false)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

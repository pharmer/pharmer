package hetzner

import (
	"bytes"
	"context"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
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
{{ end }}
`
)

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func RenderStartupScript(ctx context.Context, cluster *api.Cluster, role string) (string, error) {
	tpl, err := cloud.StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	tpl, err = tpl.Parse(customTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, role, cloud.GetTemplateData(ctx, cluster)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

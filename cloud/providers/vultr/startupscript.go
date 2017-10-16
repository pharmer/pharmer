package vultr

import (
	"bytes"
	"context"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
)

var (
	customTemplate = `
{{ define "prepare-host" }}
# https://www.vultr.com/docs/configuring-private-network
PRIVATE_ADDRESS=$(/usr/bin/curl http://169.254.169.254/v1/interfaces/1/ipv4/address 2> /dev/null)
PRIVATE_NETMASK=$(/usr/bin/curl http://169.254.169.254/v1/interfaces/1/ipv4/netmask 2> /dev/null)
/bin/cat >>/etc/network/interfaces <<EOF

auto eth1
iface eth1 inet static
    address $PRIVATE_ADDRESS
    netmask $PRIVATE_NETMASK
            mtu 1450
EOF
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

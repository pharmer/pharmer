package packet

import (
	"bytes"
	"context"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
)

var (
	customTemplate = `
{{ define "prepare-host" }}
# /bin/cat >/etc/apt/sources.list <<EOF
# deb http://ftp.us.debian.org/debian jessie main
# deb http://security.debian.org/ jessie/updates main
# deb http://ftp.us.debian.org/debian jessie-updates main
# EOF
/usr/bin/apt-get update
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

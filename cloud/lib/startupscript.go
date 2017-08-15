package lib

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/net/httpclient"
	"github.com/appscode/pharmer/api"
	"github.com/golang/protobuf/jsonpb"
)

// This is called from a /etc/rc.local script, so always use full path for any command
func RenderKubeStarter(opt *api.ScriptOptions, sku, cmd string) string {
	return fmt.Sprintf(`#!/bin/bash -e
set -o errexit
set -o nounset
set -o pipefail

export LC_ALL=en_US.UTF-8
export LANG=en_US.UTF-8
/usr/bin/apt-get update || true
/usr/bin/apt-get install -y wget curl aufs-tools

%v

/usr/bin/wget %v -O start-kubernetes
/bin/chmod a+x start-kubernetes
/bin/echo $CONFIG | ./start-kubernetes --v=3 --sku=%v
/bin/rm start-kubernetes
`,
		cmd, opt.KubeStarterURL, sku)
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func RenderKubeInstaller(opt *api.ScriptOptions, sku, role, cmd string) string {
	return fmt.Sprintf(`#!/bin/bash
cat >/etc/kube-installer.sh <<EOF
%v
rm /lib/systemd/system/kube-installer.service
systemctl daemon-reload
exit 0
EOF
chmod +x /etc/kube-installer.sh

cat >/lib/systemd/system/kube-installer.service <<EOF
[Unit]
Description=Install Kubernetes Master

[Service]
Type=simple
ExecStart=/bin/bash -e /etc/kube-installer.sh
Restart=on-failure
StartLimitInterval=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable kube-installer.service
`, strings.Replace(RenderKubeStarter(opt, sku, cmd), "$", "\\$", -1))
}

func SaveInstancesInFirebase(opt *api.ScriptOptions, ins *api.ClusterInstances) error {
	// TODO: FixIt
	// ins.Logger().Infof("Server is configured to skip startup config api")
	// store instances
	for _, v := range ins.Instances {
		if v.ExternalIP != "" {
			fbPath, err := firebaseInstancePath(opt, v.ExternalIP)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}
			fmt.Println(fbPath)

			r2 := &proto.ClusterInstanceByIPResponse{
				Instance: &proto.ClusterInstance{
					Phid:       v.PHID,
					ExternalId: v.ExternalID,
					Name:       v.Name,
					ExternalIp: v.ExternalIP,
					InternalIp: v.InternalIP,
					Sku:        v.SKU,
				},
			}

			var buf bytes.Buffer
			m := jsonpb.Marshaler{}
			err = m.Marshal(&buf, r2)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}

			_, err = httpclient.New(nil, nil, nil).
				WithBaseURL(firebaseEndpoint).
				Call(http.MethodPut, fbPath, &buf, nil, false)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}
		}
	}
	return nil
}

func UploadStartupConfigInFirebase(ctx *api.Cluster) error {
	ctx.Logger().Infof("Server is configured to skip startup config api")
	{
		cfg, err := ctx.StartupConfigResponse(api.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fbPath, err := firebaseStartupConfigPath(ctx.NewScriptOptions(), api.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fmt.Println(fbPath)

		_, err = httpclient.New(nil, nil, nil).
			WithBaseURL(firebaseEndpoint).
			Call(http.MethodPut, fbPath, bytes.NewBufferString(cfg), nil, false)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
	}
	{
		// store startup config
		cfg, err := ctx.StartupConfigResponse(api.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fbPath, err := firebaseStartupConfigPath(ctx.NewScriptOptions(), api.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fmt.Println(fbPath)

		_, err = httpclient.New(nil, nil, nil).
			WithBaseURL(firebaseEndpoint).
			Call(http.MethodPut, fbPath, bytes.NewBufferString(cfg), nil, false)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
	}
	return nil
}

func StartupConfigFromFirebase(opt *api.ScriptOptions, role string) string {
	url, _ := firebaseStartupConfigPath(opt, role)
	return fmt.Sprintf(`CONFIG=$(/usr/bin/wget -qO- '%v' 2> /dev/null)`, url)
}

func StartupConfigFromAPI(opt *api.ScriptOptions, role string) string {
	// TODO(tamal): Use wget instead of curl
	return fmt.Sprintf(`CONFIG=$(/usr/bin/wget -qO- '%v/kubernetes/v1beta1/clusters/%v/startup-script/%v/context-versions/%v/json' --header='Authorization: Bearer %v:%v' 2> /dev/null)`,
		"", // system.PublicAPIHttpEndpoint(),
		opt.PHID,
		role,
		opt.Namespace,
		opt.ContextVersion,
		opt.StartupConfigToken)
}

const firebaseEndpoint = "https://tigerworks-kube.firebaseio.com"

func firebaseStartupConfigPath(opt *api.ScriptOptions, role string) (string, error) {
	l, err := api.FirebaseUid()
	if err != nil {
		return "", errors.FromErr(err).WithContext(opt.Ctx).Err()
	}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/startup-script/%v/context-versions/%v.json?auth=%v`,
		l,
		opt.Namespace,
		opt.Name, // phid is grpc api
		role,
		opt.ContextVersion,
		opt.StartupConfigToken), nil
}

func firebaseInstancePath(opt *api.ScriptOptions, externalIP string) (string, error) {
	l, err := api.FirebaseUid()
	if err != nil {
		return "", errors.FromErr(err).WithContext(opt.Ctx).Err()
	}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/instance-by-ip/%v.json?auth=%v`,
		l,
		opt.Namespace,
		opt.Name, // phid is grpc api
		strings.Replace(externalIP, ".", "_", -1),
		opt.StartupConfigToken), nil
}

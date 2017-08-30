package aws

import (
	"bytes"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) RenderMasterStarter() (string, error) {
	tpl, err := cloud.StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	/*
		// 2. Define T2, version B, and parse it.
		_, err = second.Parse("{{define `T2`}}T2, version B{{end}}")
		if err != nil {
		        log.Fatal("parsing T2: ", err)
		}
	*/
	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, api.RoleKubernetesMaster, cloud.GetTemplateData(cm.ctx, cm.cluster))
	if err != nil {
		return "", err
	}
	out := buf.String()
	return out, nil
}

func (cm *ClusterManager) RenderNodeStarter() (string, error) {
	tpl, err := cloud.StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	/*
		// 2. Define T2, version B, and parse it.
		_, err = second.Parse("{{define `T2`}}T2, version B{{end}}")
		if err != nil {
		        log.Fatal("parsing T2: ", err)
		}
	*/
	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, api.RoleKubernetesPool, cloud.GetTemplateData(cm.ctx, cm.cluster))
	if err != nil {
		return "", err
	}
	out := buf.String()
	return out, nil
}

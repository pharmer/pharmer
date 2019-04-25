package framework

import (
	"context"

	. "github.com/onsi/gomega"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/store"
)

type Framework struct {
	Storage store.Interface
	Config  *api.PharmerConfig

	ClusterName string
}

func New(configFile string) *Framework {
	conf, err := config.LoadConfig(configFile)
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		Storage: cloud.NewStoreProvider(context.Background(), conf),
		Config:  conf,
	}
}

type Invocation struct {
	*rootInvocation
	Credential *credentialInvocation
	Cluster    *clusterInvocation
	SSH        *sshInvocation
	NG         *nodeGroupInvocaton
}

func (f *Framework) Invoke() *Invocation {
	r := &rootInvocation{
		Framework: f,
	}
	return &Invocation{
		rootInvocation: r,
		Credential:     &credentialInvocation{rootInvocation: r},
		Cluster:        &clusterInvocation{rootInvocation: r},
		SSH:            &sshInvocation{rootInvocation: r, clusterName: f.ClusterName},
		NG:             &nodeGroupInvocaton{rootInvocation: r, clusterName: f.ClusterName},
		//app:       rand.WithUniqSuffix("storage"),
	}
}

type rootInvocation struct {
	*Framework
}

type credentialInvocation struct {
	*rootInvocation
}

type clusterInvocation struct {
	*rootInvocation
}

type sshInvocation struct {
	*rootInvocation
	clusterName string
}

type nodeGroupInvocaton struct {
	*rootInvocation
	clusterName string
}

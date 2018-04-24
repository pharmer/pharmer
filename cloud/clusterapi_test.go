package cloud

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	_env "github.com/appscode/go/env"
	"github.com/pharmer/pharmer/config"
	"k8s.io/client-go/util/homedir"
)

func TestCreateApiserver(t *testing.T) {

	cfgFile := filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default")
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Println(err)
	}
	ctx := NewContext(context.Background(), cfg, _env.Dev)

	cluster, err := Store(ctx).Clusters().Get("doc2")
	fmt.Println(err)

	if ctx, err = LoadCACertificates(ctx, cluster); err != nil {
		fmt.Println(err, "----")
	}
	if ctx, err = LoadSSHKey(ctx, cluster); err != nil {
		fmt.Println(err)
	}
	/*if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return err
	}*/

	kc, err := NewAdminClient(ctx, cluster)
	if err != nil {
		fmt.Println(err)
	}

	ca, err := NewClusterApi(ctx, cluster, kc)
	if err != nil {
		fmt.Println(err)
	}
	if err := ca.Apply(); err != nil {
		fmt.Println(err)
	}

}

package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	"github.com/appscode/go/crypto/ssh"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/log"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/providers/fake"
	"github.com/appscode/pharmer/store/providers/vfs"
)

type paramEnv struct{}
type paramExtra struct{}
type paramLogger struct{}
type paramStore struct{}

type paramCACert struct{}
type paramCAKey struct{}
type paramFrontProxyCACert struct{}
type paramFrontProxyCAKey struct{}

type paramSSHKey struct{}

type paramK8sClient struct{}

func Env(ctx context.Context) _env.Environment {
	return ctx.Value(paramEnv{}).(_env.Environment)
}

func Store(ctx context.Context) store.Interface {
	return ctx.Value(paramStore{}).(store.Interface)
}

func Logger(ctx context.Context) api.Logger {
	return ctx.Value(paramLogger{}).(api.Logger)
}

func Extra(ctx context.Context) api.NameGenerator {
	return ctx.Value(paramExtra{}).(api.NameGenerator)
}

func CACert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramCACert{}).(*x509.Certificate)
}
func CAKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramCAKey{}).(*rsa.PrivateKey)
}

func FrontProxyCACert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramFrontProxyCACert{}).(*x509.Certificate)
}
func FrontProxyCAKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramFrontProxyCAKey{}).(*rsa.PrivateKey)
}

func SSHKey(ctx context.Context) *ssh.SSHKey {
	return ctx.Value(paramSSHKey{}).(*ssh.SSHKey)
}

func NewContext(parent context.Context, cfg *api.PharmerConfig, env _env.Environment) context.Context {
	c := parent
	c = context.WithValue(c, paramEnv{}, env)
	c = context.WithValue(c, paramExtra{}, &api.NullNameGenerator{})
	c = context.WithValue(c, paramLogger{}, log.New(c))
	c = context.WithValue(c, paramStore{}, NewStoreProvider(parent, cfg))
	return c
}

func NewStoreProvider(ctx context.Context, cfg *api.PharmerConfig) store.Interface {
	if store, err := store.GetProvider(vfs.UID, ctx, cfg); err == nil {
		return store
	}
	return &fake.FakeStore{}
}

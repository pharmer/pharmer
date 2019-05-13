package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	"github.com/appscode/go/crypto/ssh"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/machinesetup"
	"github.com/pharmer/pharmer/store"
	"github.com/pharmer/pharmer/store/providers/fake"
	"github.com/pharmer/pharmer/store/providers/vfs"
	"github.com/pharmer/pharmer/store/providers/xorm"
)

type paramEnv struct{}
type paramNameGen struct{}
type paramLogger struct{}
type paramStore struct{}

type paramCACert struct{}
type paramCAKey struct{}
type paramFrontProxyCACert struct{}
type paramFrontProxyCAKey struct{}

type paramApiServerCaCert struct{}
type paramApiServerCaKey struct{}
type paramApiServerCert struct{}
type paramApiServerKey struct{}

type paramEtcdCACert struct{}
type paramEtcdCAKey struct{}

type paramSSHKey struct{}

type paramK8sClient struct{}

type paramSaKey struct{}
type paramSaCert struct{}

type paramMachineSetup struct{}

func Env(ctx context.Context) _env.Environment {
	return ctx.Value(paramEnv{}).(_env.Environment)
}

func WithEnv(parent context.Context, v _env.Environment) context.Context {
	return context.WithValue(parent, paramEnv{}, v)
}

func Store(ctx context.Context) store.Interface {
	return ctx.Value(paramStore{}).(store.Interface)
}

func WithStore(parent context.Context, v store.Interface) context.Context {
	if v == nil {
		panic("nil store")
	}
	return context.WithValue(parent, paramStore{}, v)
}

func MachineSetup(ctx context.Context) *machinesetup.ConfigWatch {
	return ctx.Value(paramMachineSetup{}).(*machinesetup.ConfigWatch)
}

func WithMachineSetup(parent context.Context, v *machinesetup.ConfigWatch) context.Context {
	if v == nil {
		panic("nil machine setup")
	}
	return context.WithValue(parent, paramMachineSetup{}, v)
}

func Logger(ctx context.Context) api.Logger {
	return ctx.Value(paramLogger{}).(api.Logger)
}

func WithLogger(parent context.Context, v api.Logger) context.Context {
	if v == nil {
		panic("nil logger")
	}
	return context.WithValue(parent, paramLogger{}, v)
}

func NameGenerator(ctx context.Context) api.NameGenerator {
	return ctx.Value(paramNameGen{}).(api.NameGenerator)
}

func WithNameGenerator(parent context.Context, v api.NameGenerator) context.Context {
	if v == nil {
		panic("nil name generator")
	}
	return context.WithValue(parent, paramNameGen{}, v)
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

func ApiServerCaCert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramApiServerCaCert{}).(*x509.Certificate)
}

func ApiServerCaKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramApiServerCaKey{}).(*rsa.PrivateKey)
}

func ApiServerCert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramApiServerCert{}).(*x509.Certificate)
}

func ApiServerKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramApiServerKey{}).(*rsa.PrivateKey)
}

func EtcdCaCert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramEtcdCACert{}).(*x509.Certificate)
}

func EtcdCaKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramEtcdCAKey{}).(*rsa.PrivateKey)
}

func SSHKey(ctx context.Context) *ssh.SSHKey {
	return ctx.Value(paramSSHKey{}).(*ssh.SSHKey)
}

func SaKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramSaKey{}).(*rsa.PrivateKey)
}

func SaCert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramSaCert{}).(*x509.Certificate)
}

func NewContext(parent context.Context, cfg *api.PharmerConfig, env _env.Environment) context.Context {
	c := parent
	c = WithEnv(c, env)
	c = WithNameGenerator(c, &api.NullNameGenerator{})
	c = WithLogger(c, log.New(c))
	c = WithStore(c, NewStoreProvider(parent, cfg))
	return c
}

func NewStoreProvider(ctx context.Context, cfg *api.PharmerConfig) store.Interface {
	var storeType string
	if cfg.Store.Local != nil ||
		cfg.Store.S3 != nil ||
		cfg.Store.GCS != nil ||
		cfg.Store.Azure != nil ||
		cfg.Store.Swift != nil {
		storeType = vfs.UID
	} else if cfg.Store.Postgres != nil {
		storeType = xorm.UID
	} else {
		storeType = fake.UID
	}
	store, err := store.GetProvider(storeType, ctx, cfg)
	if err != nil {
		panic(err)
	}
	return store
}

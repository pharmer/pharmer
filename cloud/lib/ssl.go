package lib

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/system"
	"github.com/cloudflare/cfssl/cli"
	"github.com/cloudflare/cfssl/cli/genkey"
	"github.com/cloudflare/cfssl/cli/sign"
	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/cloudflare/cfssl/signer"
)

// Returns PHID, cert []byte, key []byte, error
func CreateCA(ctx *contexts.ClusterContext) (string, []byte, []byte, error) {
	var d time.Duration
	d = 10 * 365 * 24 * time.Hour
	certReq := &csr.CertificateRequest{
		CN: system.ClusterCAName(ctx.Auth.Namespace, ctx.Name),
		Hosts: []string{
			"127.0.0.1",
		},
		KeyRequest: &csr.BasicKeyRequest{A: "rsa", S: 2048},
		CA: &csr.CAConfig{
			PathLength: 2,
			Expiry:     d.String(),
		},
	}

	cert, _, key, err := initca.New(certReq)
	if err != nil {
		return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	phid := ""
	// TODO(tamal): Fix
	//phid, err := ctx.Store.InsertCertificate(cert, key, system.CertRoot, "")
	//if err != nil {
	//	return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	//}
	return phid, cert, key, nil
}

func CreateClientCert(ctx *contexts.ClusterContext, caCert, caKey []byte, csrReq *csr.CertificateRequest) (string, []byte, []byte, error) {
	g := &csr.Generator{Validator: genkey.Validator}
	csrPem, key, err := g.ProcessRequest(csrReq)
	if err != nil {
		return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	tempDir, _ := ioutil.TempDir(os.TempDir(), "cfssl")
	defer os.RemoveAll(tempDir)

	var cfg cli.Config
	cfg.CAKeyFile = tempDir + "/server.key"
	cfg.CAFile = tempDir + "/server.crt"
	cfg.CFG = &config.Config{
		Signing: &config.Signing{
			Profiles: map[string]*config.SigningProfile{},
			Default:  config.DefaultConfig(),
		},
	}
	var d time.Duration
	d = 10 * 365 * 24 * time.Hour
	cfg.CFG.Signing.Default.Expiry = d
	cfg.CFG.Signing.Default.ExpiryString = d.String()

	err = ioutil.WriteFile(cfg.CAFile, caCert, 0644)
	if err != nil {
		return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	err = ioutil.WriteFile(cfg.CAKeyFile, caKey, 0600)
	if err != nil {
		return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	s, err := sign.SignerFromConfig(cfg)
	if err != nil {
		return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	var cert []byte
	signReq := signer.SignRequest{
		Request: string(csrPem),
		Hosts:   signer.SplitHosts(cfg.Hostname),
		Profile: cfg.Profile,
		Label:   cfg.Label,
	}

	cert, err = s.Sign(signReq)
	if err != nil {
		return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	// TODO(tamal): Fix
	//phid, err := ctx.Store.InsertCertificate(cert, key, system.CertLeaf, "")
	//if err != nil {
	//	return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	//}
	phid := ""
	return phid, cert, key, nil
}

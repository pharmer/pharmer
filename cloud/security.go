package cloud

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	mrnd "math/rand"
	"os"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
	"github.com/cloudflare/cfssl/cli"
	"github.com/cloudflare/cfssl/cli/genkey"
	"github.com/cloudflare/cfssl/cli/sign"
	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/cloudflare/cfssl/signer"
)

func GetRandomToken() string {
	return fmt.Sprintf("%s.%s", RandStringRunes(6), RandStringRunes(16))
}

func init() {
	mrnd.Seed(time.Now().UnixNano())
}

// Hexidecimal
var letterRunes = []rune("0123456789abcdef")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrnd.Intn(len(letterRunes))]
	}
	return string(b)
}

func GenClusterTokens(cluster *api.Cluster) {
	cluster.KubeBearerToken = rand.GenerateToken()
	cluster.KubeletToken = rand.GenerateToken()
	cluster.KubeProxyToken = rand.GenerateToken()
}

func GenClusterCerts(ctx context.Context, cluster *api.Cluster) error {
	ctx.Logger().Infoln("Generating certificate for cluster")

	var csrReq csr.CertificateRequest
	csrReq.KeyRequest = &csr.BasicKeyRequest{A: "rsa", S: 2048}

	////////// Cluster CA //////////
	//caCN := system.ClusterCAName(cluster.Auth.Namespace, cluster.Name)
	var caCertPHID string
	var caCert, caKey []byte

	// TODO: FixIt!
	//caRow := &storage.Certificate{
	//	CommonName: cluster.PHID,
	//}
	//has, err := cluster.Store.Engine.Get(caRow)
	//if err == nil && has {
	//	ctx.Logger().Infof("Found existing CA cert with PHID:%v", cluster.CaCertPHID)
	//
	//	caCertPHID = caRow.PHID
	//	caCert = []byte(caRow.Cert)
	//	sec, err := storage.NewSecEnvelope(caRow.Key)
	//	if err != nil {
	//		return err
	//	}
	//	caKey, err = sec.ValBytes()
	//	if err != nil {
	//		return err
	//	}
	//} else {

	var err error
	caCertPHID, caCert, caKey, err = CreateCA(ctx)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	ctx.Logger().Infof("Created CA cert with PHID:%v", caCertPHID)

	cluster.CaCertPHID = caCertPHID
	cluster.CaCert = base64.StdEncoding.EncodeToString(caCert)
	cluster.CaKey = base64.StdEncoding.EncodeToString(caKey)
	////////////////////////
	////////// Front Proxy CA ////////
	frontCACertPHID, frontCACert, frontCAKey, err := CreateCA(ctx)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	cluster.FrontProxyCaCertPHID = frontCACertPHID
	cluster.FrontProxyCaCert = base64.StdEncoding.EncodeToString(frontCACert)
	cluster.FrontProxyCaKey = base64.StdEncoding.EncodeToString(frontCAKey)

	csrReq.CN = "kubernetes-user"
	//csrReq.Hosts = []string{"127.0.0.1"}
	csrReq.Names = []csr.Name{
		{
			O: "system:masters",
		},
	}
	csrReq.KeyRequest = &csr.BasicKeyRequest{A: "rsa", S: 2048}
	userCertPHID, userCert, userKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	cluster.UserCertPHID = userCertPHID
	cluster.UserCert = base64.StdEncoding.EncodeToString(userCert)
	cluster.UserKey = base64.StdEncoding.EncodeToString(userKey)
	ctx.Logger().Infof("Created user cert %v  key %v with PHID:%v", string(userCert), string(userKey), cluster.UserCertPHID)

	ctx.Logger().Infoln("Certificates generated successfully")
	return nil
}

// Returns PHID, cert []byte, key []byte, error
func CreateCA(ctx context.Context) (string, []byte, []byte, error) {
	var d time.Duration
	d = 10 * 365 * 24 * time.Hour
	certReq := &csr.CertificateRequest{
		CN: "pharmer",
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
	//phid, err := ctx.Store().InsertCertificate(cert, key, system.CertRoot, "")
	//if err != nil {
	//	return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	//}
	return phid, cert, key, nil
}

func CreateClientCert(ctx context.Context, caCert, caKey []byte, csrReq *csr.CertificateRequest) (string, []byte, []byte, error) {
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
	//phid, err := ctx.Store().InsertCertificate(cert, key, system.CertLeaf, "")
	//if err != nil {
	//	return "", nil, nil, errors.FromErr(err).WithContext(ctx).Err()
	//}
	phid := ""
	return phid, cert, key, nil
}

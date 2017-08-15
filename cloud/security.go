package cloud

import (
	"encoding/base64"

	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
	"github.com/cloudflare/cfssl/csr"
)

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

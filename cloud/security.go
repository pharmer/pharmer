package cloud

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/appscode/pharmer/api"
	"k8s.io/client-go/util/cert"
)

func GenClusterCerts(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	Logger(ctx).Infoln("Generating certificate for cluster")

	certStore := Store(ctx).Certificates(cluster.Name)

	// -----------------------------------------------

	cluster.Spec.CACertName = "ca"

	caKey, err := cert.NewPrivateKey()
	if err != nil {
		return ctx, fmt.Errorf("Failed to generate private key. Reason: %v.", err)
	}
	caCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: cluster.Spec.CACertName}, caKey)
	if err != nil {
		return ctx, fmt.Errorf("Failed to generate self-signed certificate. Reason: %v.", err)
	}

	ctx = context.WithValue(ctx, keyCACert{}, caCert)
	ctx = context.WithValue(ctx, keyCAKey{}, caKey)
	certStore.Create(cluster.Spec.CACertName, cert.EncodeCertPEM(caCert), cert.EncodePrivateKeyPEM(caKey))

	// -----------------------------------------------

	cluster.Spec.FrontProxyCACertName = "front-proxy-ca"
	frontProxyCAKey, err := cert.NewPrivateKey()
	if err != nil {
		return ctx, fmt.Errorf("Failed to generate private key. Reason: %v.", err)
	}
	frontProxyCACert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: cluster.Spec.CACertName}, frontProxyCAKey)
	if err != nil {
		return ctx, fmt.Errorf("Failed to generate self-signed certificate. Reason: %v.", err)
	}

	ctx = context.WithValue(ctx, keyFrontProxyCACert{}, frontProxyCACert)
	ctx = context.WithValue(ctx, keyFrontProxyCAKey{}, frontProxyCAKey)
	certStore.Create(cluster.Spec.FrontProxyCACertName, cert.EncodeCertPEM(frontProxyCACert), cert.EncodePrivateKeyPEM(frontProxyCAKey))

	// -----------------------------------------------

	cluster.Spec.AdminUserCertName = "cluster-admin"
	cfg := cert.Config{
		CommonName:   cluster.Spec.AdminUserCertName,
		Organization: []string{"system:masters"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	adminUserKey, err := cert.NewPrivateKey()
	if err != nil {
		return ctx, fmt.Errorf("Failed to generate private key. Reason: %v.", err)
	}
	adminUserCert, err := cert.NewSignedCert(cfg, adminUserKey, caCert, caKey)
	if err != nil {
		return ctx, fmt.Errorf("Failed to generate server certificate. Reason: %v.", err)
	}
	ctx = context.WithValue(ctx, keyAdminUserCert{}, adminUserCert)
	ctx = context.WithValue(ctx, keyAdminUserKey{}, adminUserKey)

	Logger(ctx).Infoln("Certificates generated successfully")
	return ctx, nil
}

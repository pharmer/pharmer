package cloud

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/appscode/pharmer/api"
	"k8s.io/client-go/util/cert"
)

func GenerateCertificates(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
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

	ctx = context.WithValue(ctx, paramCACert{}, caCert)
	ctx = context.WithValue(ctx, paramCAKey{}, caKey)
	certStore.Create(cluster.Spec.CACertName, caCert, caKey)

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

	ctx = context.WithValue(ctx, paramFrontProxyCACert{}, frontProxyCACert)
	ctx = context.WithValue(ctx, paramFrontProxyCAKey{}, frontProxyCAKey)
	certStore.Create(cluster.Spec.FrontProxyCACertName, frontProxyCACert, frontProxyCAKey)

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
	ctx = context.WithValue(ctx, paramAdminUserCert{}, adminUserCert)
	ctx = context.WithValue(ctx, paramAdminUserKey{}, adminUserKey)

	Logger(ctx).Infoln("Certificates generated successfully")
	return ctx, nil
}

func GenerateSSHKey(ctx context.Context) (context.Context, error) {
	sshKey, err := api.NewSSHKeyPair()
	if err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, paramSSHKey{}, sshKey)
	return ctx, nil
}

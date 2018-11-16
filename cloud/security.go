package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	"github.com/appscode/go/crypto/ssh"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/cert"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func CreateCACertificates(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	Logger(ctx).Infoln("Generating CA certificate for cluster")

	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)

	// -----------------------------------------------
	if cluster.Spec.CACertName == "" {
		cluster.Spec.CACertName = kubeadmconst.CACertAndKeyBaseName

		caKey, err := cert.NewPrivateKey()
		if err != nil {
			return ctx, errors.Errorf("failed to generate private key. Reason: %v", err)
		}
		caCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: cluster.Spec.CACertName}, caKey)
		if err != nil {
			return ctx, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
		}

		ctx = context.WithValue(ctx, paramCACert{}, caCert)
		ctx = context.WithValue(ctx, paramCAKey{}, caKey)
		if err = certStore.Create(cluster.Spec.CACertName, caCert, caKey); err != nil {
			return ctx, err
		}
	}

	// -----------------------------------------------
	if cluster.Spec.FrontProxyCACertName == "" {
		cluster.Spec.FrontProxyCACertName = kubeadmconst.FrontProxyCACertAndKeyBaseName
		frontProxyCAKey, err := cert.NewPrivateKey()
		if err != nil {
			return ctx, errors.Errorf("failed to generate private key. Reason: %v", err)
		}
		frontProxyCACert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: cluster.Spec.CACertName}, frontProxyCAKey)
		if err != nil {
			return ctx, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
		}

		ctx = context.WithValue(ctx, paramFrontProxyCACert{}, frontProxyCACert)
		ctx = context.WithValue(ctx, paramFrontProxyCAKey{}, frontProxyCAKey)
		if err = certStore.Create(cluster.Spec.FrontProxyCACertName, frontProxyCACert, frontProxyCAKey); err != nil {
			return ctx, err
		}
	}

	Logger(ctx).Infoln("CA certificates generated successfully.")
	return ctx, nil
}

func LoadCACertificates(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)

	caCert, caKey, err := certStore.Get(cluster.Spec.CACertName)
	if err != nil {
		return ctx, errors.Errorf("failed to get CA certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramCACert{}, caCert)
	ctx = context.WithValue(ctx, paramCAKey{}, caKey)

	frontProxyCACert, frontProxyCAKey, err := certStore.Get(cluster.Spec.FrontProxyCACertName)
	if err != nil {
		return ctx, errors.Errorf("failed to get front proxy CA certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramFrontProxyCACert{}, frontProxyCACert)
	ctx = context.WithValue(ctx, paramFrontProxyCAKey{}, frontProxyCAKey)

	return ctx, nil
}

func CreateAdminCertificate(ctx context.Context) (*x509.Certificate, *rsa.PrivateKey, error) {
	cfg := cert.Config{
		CommonName:   "cluster-admin",
		Organization: []string{kubeadmconst.MastersGroup},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	adminKey, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate private key. Reason: %v", err)
	}
	adminCert, err := cert.NewSignedCert(cfg, adminKey, CACert(ctx), CAKey(ctx))
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate server certificate. Reason: %v", err)
	}
	return adminCert, adminKey, nil
}

func GetAdminCertificate(ctx context.Context, cluster *api.Cluster, owner string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)
	admCert, admKey, err := certStore.Get("admin")
	if err != nil {
		return nil, nil, errors.Errorf("failed to get admin certificates. Reason: %v", err)
	}
	return admCert, admKey, nil
}

func CreateSSHKey(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	sshKey, err := ssh.NewSSHKeyPair()
	if err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, paramSSHKey{}, sshKey)
	err = Store(ctx).Owner(owner).SSHKeys(cluster.Name).Create(cluster.Spec.Cloud.SSHKeyName, sshKey.PublicKey, sshKey.PrivateKey)
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

func LoadSSHKey(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	publicKey, privateKey, err := Store(ctx).Owner(owner).SSHKeys(cluster.Name).Get(cluster.Spec.Cloud.SSHKeyName)
	if err != nil {
		return ctx, errors.Errorf("failed to get SSH key. Reason: %v", err)
	}

	protoSSH, err := ssh.ParseSSHKeyPair(string(publicKey), string(privateKey))
	if err != nil {
		return ctx, errors.Errorf("failed to parse SSH key. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramSSHKey{}, protoSSH)
	return ctx, nil
}

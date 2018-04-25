package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/appscode/go/crypto/ssh"
	api "github.com/pharmer/pharmer/apis/v1"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/cert"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/pkg/apis/core"
)

func CreateCACertificates(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	Logger(ctx).Infoln("Generating CA certificate for cluster")

	certStore := Store(ctx).Certificates(cluster.Name)

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

func CreateApiserverCertificates(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	Logger(ctx).Infoln("Generating Apiserver certificate for cluster")

	const name = "clusterapi"
	const namespace = core.NamespaceDefault

	certStore := Store(ctx).Certificates(cluster.Name)
	// -----------------------------------------------

	// apiserver ca cert
	caKey, err := cert.NewPrivateKey()
	if err != nil {
		return ctx, errors.Errorf("failed to generate private key. Reason: %v", err)
	}
	cfg := cert.Config{
		CommonName: fmt.Sprintf("%v-certificate-authority", name),
	}
	caCert, err := cert.NewSelfSignedCACert(cfg, caKey)
	if err != nil {
		return ctx, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
	}

	ctx = context.WithValue(ctx, paramApiServerCaCert{}, caCert)
	ctx = context.WithValue(ctx, paramApiServerCaKey{}, caKey)
	if err = certStore.Create(kubeadmconst.APIServerCertAndKeyBaseName+"-ca", caCert, caKey); err != nil {
		return ctx, err
	}

	cfg = cert.Config{
		CommonName: fmt.Sprintf("%v.%v.svc", name, namespace),
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	apiServerKey, err := cert.NewPrivateKey()
	if err != nil {
		return ctx, errors.Errorf("failed to generate private key. Reason: %v", err)
	}
	apiServerCert, err := cert.NewSignedCert(cfg, apiServerKey, ApiServerCaCert(ctx), ApiServerCaKey(ctx))
	if err != nil {
		return ctx, errors.Errorf("failed to generate server certificate. Reason: %v", err)
	}

	ctx = context.WithValue(ctx, paramApiServerCert{}, apiServerCert)
	ctx = context.WithValue(ctx, paramApiServerKey{}, apiServerKey)
	if err = certStore.Create(kubeadmconst.APIServerCertAndKeyBaseName, apiServerCert, apiServerKey); err != nil {
		return ctx, err
	}
	Logger(ctx).Infoln("Apiserver certificates generated successfully.")
	return ctx, nil
}

func LoadCACertificates(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	certStore := Store(ctx).Certificates(cluster.Name)

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

func LoadApiserverCertificate(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	certStore := Store(ctx).Certificates(cluster.Name)
	apiserverCaCert, apiserverCaKey, err := certStore.Get(kubeadmconst.APIServerCertAndKeyBaseName + "-ca")
	if err != nil {
		return ctx, errors.Errorf("failed to get apiserver certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramApiServerCaCert{}, apiserverCaCert)
	ctx = context.WithValue(ctx, paramApiServerCaKey{}, apiserverCaKey)

	apiserverCert, apiserverKey, err := certStore.Get(kubeadmconst.APIServerCertAndKeyBaseName)
	if err != nil {
		return ctx, errors.Errorf("failed to get apiserver certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramApiServerCert{}, apiserverCert)
	ctx = context.WithValue(ctx, paramApiServerKey{}, apiserverKey)

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

func GetAdminCertificate(ctx context.Context, cluster *api.Cluster) (*x509.Certificate, *rsa.PrivateKey, error) {
	certStore := Store(ctx).Certificates(cluster.Name)
	admCert, admKey, err := certStore.Get("admin")
	if err != nil {
		return nil, nil, errors.Errorf("failed to get admin certificates. Reason: %v", err)
	}
	return admCert, admKey, nil
}

func CreateSSHKey(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	sshKey, err := ssh.NewSSHKeyPair()
	if err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, paramSSHKey{}, sshKey)
	err = Store(ctx).SSHKeys(cluster.Name).Create(cluster.ProviderConfig().SSHKeyName, sshKey.PublicKey, sshKey.PrivateKey)
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

func LoadSSHKey(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	fmt.Println(cluster.ProviderConfig().SSHKeyName)
	publicKey, privateKey, err := Store(ctx).SSHKeys(cluster.Name).Get(cluster.ProviderConfig().SSHKeyName)
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

package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/appscode/go/crypto/ssh"
	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/cert"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func CreateCACertificates(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	log.Infoln("Generating CA certificate for cluster")

	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)

	// -----------------------------------------------
	if cluster.Spec.Config.CACertName == "" {
		cluster.Spec.Config.CACertName = kubeadmconst.CACertAndKeyBaseName

		caKey, err := cert.NewPrivateKey()
		if err != nil {
			return ctx, errors.Errorf("failed to generate private key. Reason: %v", err)
		}
		caCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: cluster.Spec.Config.CACertName}, caKey)
		if err != nil {
			return ctx, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
		}

		ctx = context.WithValue(ctx, paramCACert{}, caCert)
		ctx = context.WithValue(ctx, paramCAKey{}, caKey)
		if err = certStore.Create(cluster.Spec.Config.CACertName, caCert, caKey); err != nil {
			return ctx, err
		}
	}

	// -----------------------------------------------
	if cluster.Spec.Config.FrontProxyCACertName == "" {
		cluster.Spec.Config.FrontProxyCACertName = kubeadmconst.FrontProxyCACertAndKeyBaseName
		frontProxyCAKey, err := cert.NewPrivateKey()
		if err != nil {
			return ctx, errors.Errorf("failed to generate private key. Reason: %v", err)
		}
		frontProxyCACert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: cluster.Spec.Config.CACertName}, frontProxyCAKey)
		if err != nil {
			return ctx, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
		}

		ctx = context.WithValue(ctx, paramFrontProxyCACert{}, frontProxyCACert)
		ctx = context.WithValue(ctx, paramFrontProxyCAKey{}, frontProxyCAKey)
		if err = certStore.Create(cluster.Spec.Config.FrontProxyCACertName, frontProxyCACert, frontProxyCAKey); err != nil {
			return ctx, err
		}
	}

	log.Infoln("CA certificates generated successfully.")
	return ctx, nil
}

func CreateServiceAccountKey(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	log.Infoln("Generating Service account signing key for cluster")
	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)

	saSigningKey, err := cert.NewPrivateKey()
	if err != nil {
		return ctx, errors.Errorf("failure while creating service account token signing key: %v", err)
	}
	cfg := cert.Config{
		CommonName: fmt.Sprintf("%v-certificate-authority", kubeadmconst.ServiceAccountKeyBaseName),
	}
	SaSigningCert, err := cert.NewSelfSignedCACert(cfg, saSigningKey)
	if err != nil {
		return ctx, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
	}

	ctx = context.WithValue(ctx, paramSaKey{}, saSigningKey)
	ctx = context.WithValue(ctx, paramSaCert{}, SaSigningCert)
	if err = certStore.Create(kubeadmconst.ServiceAccountKeyBaseName, SaSigningCert, saSigningKey); err != nil {
		return ctx, err
	}

	log.Infoln("Service account key generated successfully.")
	return ctx, nil
}

func CreateEtcdCertificates(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	log.Infoln("Generating ETCD CA certificate for etcd")

	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)

	// -----------------------------------------------
	caKey, err := cert.NewPrivateKey()
	if err != nil {
		return ctx, errors.Errorf("failed to generate private key. Reason: %v", err)
	}
	caCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "kubernetes"}, caKey)
	if err != nil {
		return ctx, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
	}

	ctx = context.WithValue(ctx, paramEtcdCACert{}, caCert)
	ctx = context.WithValue(ctx, paramEtcdCAKey{}, caKey)
	if err = certStore.Create(EtcdCACertAndKeyBaseName, caCert, caKey); err != nil {
		return ctx, err
	}

	log.Infoln("ETCD CA certificates generated successfully.")
	return ctx, nil
}

func LoadCACertificates(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)

	caCert, caKey, err := certStore.Get(cluster.Spec.Config.CACertName)
	if err != nil {
		return ctx, errors.Errorf("failed to get CA certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramCACert{}, caCert)
	ctx = context.WithValue(ctx, paramCAKey{}, caKey)

	frontProxyCACert, frontProxyCAKey, err := certStore.Get(cluster.Spec.Config.FrontProxyCACertName)
	if err != nil {
		return ctx, errors.Errorf("failed to get front proxy CA certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramFrontProxyCACert{}, frontProxyCACert)
	ctx = context.WithValue(ctx, paramFrontProxyCAKey{}, frontProxyCAKey)

	return ctx, nil
}

func LoadApiserverCertificate(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)
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

func LoadSaKey(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)
	cert, key, err := certStore.Get(kubeadmconst.ServiceAccountKeyBaseName)
	if err != nil {
		return ctx, errors.Errorf("failed to get service account key. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramSaKey{}, key)
	ctx = context.WithValue(ctx, paramSaCert{}, cert)
	return ctx, nil
}

func LoadEtcdCertificate(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	certStore := Store(ctx).Owner(owner).Certificates(cluster.Name)
	etcdCaCert, etcdCaKey, err := certStore.Get(EtcdCACertAndKeyBaseName)
	if err != nil {
		return ctx, errors.Errorf("failed to get etcd certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramEtcdCACert{}, etcdCaCert)
	ctx = context.WithValue(ctx, paramEtcdCAKey{}, etcdCaKey)

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
	err = Store(ctx).Owner(owner).SSHKeys(cluster.Name).Create(cluster.Spec.Config.Cloud.SSHKeyName, sshKey.PublicKey, sshKey.PrivateKey)
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

func LoadSSHKey(ctx context.Context, cluster *api.Cluster, owner string) (context.Context, error) {
	publicKey, privateKey, err := Store(ctx).Owner(owner).SSHKeys(cluster.Name).Get(cluster.Spec.Config.Cloud.SSHKeyName)
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

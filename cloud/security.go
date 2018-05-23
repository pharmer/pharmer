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
	"k8s.io/client-go/util/cert/triple"
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

func CreateServiceAccountKey(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	Logger(ctx).Infoln("Generating Service account signing key for cluster")
	certStore := Store(ctx).Certificates(cluster.Name)

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
	if err = certStore.Create(kubeadmconst.ServiceAccountKeyBaseName, SaSigningCert, saSigningKey); err != nil {
		return ctx, err
	}

	Logger(ctx).Infoln("Service account key generated successfully.")
	return ctx, nil
}

func CreateApiserverCertificates(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	Logger(ctx).Infoln("Generating Apiserver certificate for cluster")

	const name = "clusterapi"
	const namespace = core.NamespaceDefault

	certStore := Store(ctx).Certificates(cluster.Name)
	// -----------------------------------------------

	caKeyPair, err := triple.NewCA(fmt.Sprintf("%s-certificate-authority", name))
	if err != nil {
		return nil, fmt.Errorf("failed to create root-ca: %v", err)
	}

	ctx = context.WithValue(ctx, paramApiServerCaCert{}, caKeyPair.Cert)
	ctx = context.WithValue(ctx, paramApiServerCaKey{}, caKeyPair.Key)
	if err = certStore.Create(kubeadmconst.APIServerCertAndKeyBaseName+"-ca", caKeyPair.Cert, caKeyPair.Key); err != nil {
		return ctx, err
	}

	apiServerKeyPair, err := triple.NewServerKeyPair(caKeyPair,
		fmt.Sprintf("%s.%s.svc", name, namespace),
		name,
		namespace,
		"cluster.local",
		[]string{},
		[]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to create apisrver cert: %v", err)
	}

	ctx = context.WithValue(ctx, paramApiServerCert{}, apiServerKeyPair.Cert)
	ctx = context.WithValue(ctx, paramApiServerKey{}, apiServerKeyPair.Key)
	if err = certStore.Create(kubeadmconst.APIServerCertAndKeyBaseName, apiServerKeyPair.Cert, apiServerKeyPair.Key); err != nil {
		return ctx, err
	}
	Logger(ctx).Infoln("Apiserver certificates generated successfully.")
	return ctx, nil
}

func CreateEtcdCertificates(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	Logger(ctx).Infoln("Generating ETCD CA certificate for etcd")

	certStore := Store(ctx).Certificates(cluster.Name)

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

	Logger(ctx).Infoln("ETCD CA certificates generated successfully.")
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

func LoadSaKey(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	certStore := Store(ctx).Certificates(cluster.Name)
	_, key, err := certStore.Get(kubeadmconst.ServiceAccountKeyBaseName)
	if err != nil {
		return ctx, errors.Errorf("failed to get service account key. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramSaKey{}, key)
	return ctx, nil
}

func LoadEtcdCertificate(ctx context.Context, cluster *api.Cluster) (context.Context, error) {
	certStore := Store(ctx).Certificates(cluster.Name)
	etcdCaCert, etcdCaKey, err := certStore.Get(EtcdCACertAndKeyBaseName)
	if err != nil {
		return ctx, errors.Errorf("failed to get etcd certificates. Reason: %v", err)
	}
	ctx = context.WithValue(ctx, paramEtcdCACert{}, etcdCaCert)
	ctx = context.WithValue(ctx, paramEtcdCAKey{}, etcdCaKey)

	return ctx, nil
}

func CreateEtcdServerCertAndKey() {

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

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
	ctx.Logger().Info("Generating certificate for cluster")

	var csrReq csr.CertificateRequest
	csrReq.KeyRequest = &csr.BasicKeyRequest{A: "rsa", S: 2048}

	////////// Cluster CA //////////
	var caCertPHID string
	var caCert, caKey []byte

	// TODO(tamal): FixIt!
	//caRow := &storage.Certificate{
	//	CommonName: ctx.PHID,
	//}
	//has, err := ctx.Store().Engine.Get(caRow)
	//if err == nil && has {
	//	ctx.Logger().Infof("Found existing CA cert with PHID:%v", ctx.CaCertPHID)
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
	//	caCertPHID, caCert, caKey, err = CreateCA(ctx)
	//	if err != nil {
	//		return errors.FromErr(err).WithContext(ctx).Err()
	//	}
	//	ctx.Logger().Infof("Created CA cert with PHID:%v", ctx.CaCertPHID)
	//}
	cluster.CaCertPHID = caCertPHID
	cluster.CaCert = base64.StdEncoding.EncodeToString(caCert)
	////////////////////////

	////////// Master ////////////
	csrReq.CN = cluster.KubernetesMasterName
	// TODO: refactor MES generation via lib function in provider/cloud.go
	// Pass *sql object
	csrReq.Hosts = []string{
		cluster.KubernetesClusterIP(), // 10.0.0.1
		"kubernetes",
		"api.default",
		"api.default.svc",
		"api.default.svc." + cluster.DNSDomain,
		cluster.KubernetesMasterName,
		ctx.Extra().ExternalDomain(cluster.Name),
		ctx.Extra().InternalDomain(cluster.Name),
	}
	if cluster.MasterReservedIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, cluster.MasterReservedIP)
	} else if cluster.MasterExternalIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, cluster.MasterExternalIP)
	}
	if cluster.MasterInternalIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, cluster.MasterInternalIP)
	}
	ctx.Logger().Infof("Master Extra SANS: %v", csrReq.Hosts)

	masterCertPHID, masterCert, masterKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	cluster.MasterCertPHID = masterCertPHID
	cluster.MasterCert = base64.StdEncoding.EncodeToString(masterCert)
	cluster.MasterKey = base64.StdEncoding.EncodeToString(masterKey)
	ctx.Logger().Infof("Created master cert %v with PHID:%v", string(cluster.MasterCert), cluster.MasterCertPHID)
	//////////////////////////////

	////////// Default LB ////////////
	csrReq.CN = ctx.Extra().ExternalDomain(cluster.Name) //cluster-ns.appscode.(tools | tech)
	csrReq.Hosts = []string{"127.0.0.1"}
	if cluster.MasterReservedIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, cluster.MasterReservedIP)
	}
	ctx.Logger().Infof("Master LB Extra SANS: %v", csrReq.Hosts)

	defaultLBCertPHID, defaultLBCert, defaultLBKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	cluster.DefaultLBCertPHID = defaultLBCertPHID
	cluster.DefaultLBCert = base64.StdEncoding.EncodeToString(defaultLBCert)
	cluster.DefaultLBKey = base64.StdEncoding.EncodeToString(defaultLBKey)
	ctx.Logger().Infof("Created Default LB cert %v with PHID:%v", string(cluster.DefaultLBCert), cluster.DefaultLBCertPHID)
	//////////////////////////////

	////////// Kubelet //////////
	csrReq.CN = "kubelet"
	csrReq.Hosts = []string{"127.0.0.1"}
	kubeletCertPHID, kubeletCert, kubeletKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	cluster.KubeletCertPHID = kubeletCertPHID
	cluster.KubeletCert = base64.StdEncoding.EncodeToString(kubeletCert)
	cluster.KubeletKey = base64.StdEncoding.EncodeToString(kubeletKey)
	ctx.Logger().Infof("Created kubelet cert %v with PHID:%v", string(cluster.KubeletCert), cluster.KubeletCertPHID)
	/////////////////////////////

	if cluster.EnableClusterVPN == "h2h-psk" {
		////////// Kube api server //////////
		csrReq.CN = "system:apiserver"
		csrReq.Hosts = []string{"127.0.0.1"}
		kubeAPIServerPHID, kubeAPIServerCert, kubeAPIServerKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		cluster.KubeAPIServerCertPHID = kubeAPIServerPHID
		cluster.KubeAPIServerCert = base64.StdEncoding.EncodeToString(kubeAPIServerCert)
		cluster.KubeAPIServerKey = base64.StdEncoding.EncodeToString(kubeAPIServerKey)
		ctx.Logger().Infof("Created kube apiserver cert %v with PHID:%v", string(cluster.KubeAPIServerCert), cluster.KubeAPIServerCertPHID)
		//////////////////////////////////

		////////// Hostfacts server //////////
		csrReq.CN = "hostfatcs"
		csrReq.Hosts = []string{"127.0.0.1"}
		hostfactsPHID, hostfactsCert, hostfactsKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		cluster.HostfactsCertPHID = hostfactsPHID
		cluster.HostfactsCert = base64.StdEncoding.EncodeToString(hostfactsCert)
		cluster.HostfactsKey = base64.StdEncoding.EncodeToString(hostfactsKey)
		cluster.HostfactsAuthToken = rand.GenerateToken()
		ctx.Logger().Infof("Created hostfacts cert %v with PHID:%v", string(cluster.HostfactsCert), cluster.HostfactsCertPHID)
		//////////////////////////////////
	}
	ctx.Logger().Info("Certificates generated successfully")
	return nil
}

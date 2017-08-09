package common

import (
	"encoding/base64"
	"fmt"

	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/system"
	"github.com/cloudflare/cfssl/csr"
)

func GenClusterTokens(ctx *contexts.ClusterContext) {
	ctx.KubeBearerToken = rand.GenerateToken()
	ctx.KubeletToken = rand.GenerateToken()
	ctx.KubeProxyToken = rand.GenerateToken()
	ctx.KubeUser = system.ClusterUsername(ctx.Auth.Namespace, ctx.Name, "admin")
	ctx.KubePassword = rand.GeneratePassword()
}

func GenClusterCerts(ctx *contexts.ClusterContext) error {
	ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Generating certificate for cluster")

	var csrReq csr.CertificateRequest
	csrReq.KeyRequest = &csr.BasicKeyRequest{A: "rsa", S: 2048}

	////////// Cluster CA //////////
	//caCN := system.ClusterCAName(ctx.Auth.Namespace, ctx.Name)
	var caCertPHID string
	var caCert, caKey []byte

	// TODO(tamal): FixIt!
	//caRow := &storage.Certificate{
	//	CommonName: ctx.PHID,
	//}
	//has, err := ctx.Store.Engine.Get(caRow)
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
	ctx.CaCertPHID = caCertPHID
	ctx.CaCert = base64.StdEncoding.EncodeToString(caCert)
	////////////////////////

	////////// Master ////////////
	csrReq.CN = ctx.KubernetesMasterName
	// TODO: refactor MES generation via lib function in provider/lib.go
	// Pass *sql object
	csrReq.Hosts = []string{
		ctx.KubernetesClusterIP(), // 10.0.0.1
		"kubernetes",
		"api.default",
		"api.default.svc",
		"api.default.svc." + ctx.DNSDomain,
		ctx.KubernetesMasterName,
		system.ClusterExternalDomain(ctx.Auth.Namespace, ctx.Name),
		system.ClusterInternalDomain(ctx.Auth.Namespace, ctx.Name),
	}
	if ctx.MasterReservedIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, ctx.MasterReservedIP)
	} else if ctx.MasterExternalIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, ctx.MasterExternalIP)
	}
	if ctx.MasterInternalIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, ctx.MasterInternalIP)
	}
	ctx.Logger().Infof("Master Extra SANS: %v", csrReq.Hosts)

	masterCertPHID, masterCert, masterKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	ctx.MasterCertPHID = masterCertPHID
	ctx.MasterCert = base64.StdEncoding.EncodeToString(masterCert)
	ctx.MasterKey = base64.StdEncoding.EncodeToString(masterKey)
	ctx.Logger().Infof("Created master cert %v with PHID:%v", string(ctx.MasterCert), ctx.MasterCertPHID)
	//////////////////////////////

	////////// Default LB ////////////
	csrReq.CN = system.ClusterExternalDomain(ctx.Auth.Namespace, ctx.Name) //cluster-ns.appscode.(tools | tech)
	csrReq.Hosts = []string{"127.0.0.1"}
	if ctx.MasterReservedIP != "" {
		csrReq.Hosts = append(csrReq.Hosts, ctx.MasterReservedIP)
	}
	ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Master LB Extra SANS: %v", csrReq.Hosts))

	defaultLBCertPHID, defaultLBCert, defaultLBKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	ctx.DefaultLBCertPHID = defaultLBCertPHID
	ctx.DefaultLBCert = base64.StdEncoding.EncodeToString(defaultLBCert)
	ctx.DefaultLBKey = base64.StdEncoding.EncodeToString(defaultLBKey)
	ctx.Logger().Infof("Created Default LB cert %v with PHID:%v", string(ctx.DefaultLBCert), ctx.DefaultLBCertPHID)
	//////////////////////////////

	////////// Kubelet //////////
	csrReq.CN = "kubelet"
	csrReq.Hosts = []string{"127.0.0.1"}
	kubeletCertPHID, kubeletCert, kubeletKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}
	ctx.KubeletCertPHID = kubeletCertPHID
	ctx.KubeletCert = base64.StdEncoding.EncodeToString(kubeletCert)
	ctx.KubeletKey = base64.StdEncoding.EncodeToString(kubeletKey)
	ctx.Logger().Infof("Created kubelet cert %v with PHID:%v", string(ctx.KubeletCert), ctx.KubeletCertPHID)
	/////////////////////////////

	if ctx.EnableClusterVPN == "h2h-psk" {
		////////// Kube api server //////////
		csrReq.CN = "system:apiserver"
		csrReq.Hosts = []string{"127.0.0.1"}
		kubeAPIServerPHID, kubeAPIServerCert, kubeAPIServerKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		ctx.KubeAPIServerCertPHID = kubeAPIServerPHID
		ctx.KubeAPIServerCert = base64.StdEncoding.EncodeToString(kubeAPIServerCert)
		ctx.KubeAPIServerKey = base64.StdEncoding.EncodeToString(kubeAPIServerKey)
		ctx.Logger().Infof("Created kube apiserver cert %v with PHID:%v", string(ctx.KubeAPIServerCert), ctx.KubeAPIServerCertPHID)
		//////////////////////////////////

		////////// Hostfacts server //////////
		csrReq.CN = "hostfatcs"
		csrReq.Hosts = []string{"127.0.0.1"}
		hostfactsPHID, hostfactsCert, hostfactsKey, err := CreateClientCert(ctx, caCert, caKey, &csrReq)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		ctx.HostfactsCertPHID = hostfactsPHID
		ctx.HostfactsCert = base64.StdEncoding.EncodeToString(hostfactsCert)
		ctx.HostfactsKey = base64.StdEncoding.EncodeToString(hostfactsKey)
		ctx.HostfactsAuthToken = rand.GenerateToken()
		ctx.Logger().Infof("Created hostfacts cert %v with PHID:%v", string(ctx.HostfactsCert), ctx.HostfactsCertPHID)
		//////////////////////////////////
	}
	ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Certificates generated successfully")
	return nil
}

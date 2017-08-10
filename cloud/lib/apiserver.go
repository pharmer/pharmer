package lib

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/system"
	"github.com/cenkalti/backoff"
	"github.com/olekukonko/tablewriter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func EnsureARecord(ctx *contexts.ClusterContext, master *contexts.KubernetesInstance) error {
	clusterDomain := ctx.Extra.Domain(ctx.Name)
	for _, ip := range system.Config.Compass.IPs {
		if err := ctx.DNSProvider.EnsureARecord(clusterDomain, ip); err != nil {
			return err
		}
	}
	ctx.Logger.Infof("Cluster apps A record %v added", clusterDomain)
	externalDomain := ctx.Extra.ExternalDomain(ctx.Name)
	if err := ctx.DNSProvider.EnsureARecord(externalDomain, master.ExternalIP); err != nil {
		return err
	}
	ctx.Logger.Infof("External A record %v added", externalDomain)
	internalDomain := ctx.Extra.InternalDomain(ctx.Name)
	if err := ctx.DNSProvider.EnsureARecord(internalDomain, master.InternalIP); err != nil {
		return err
	}
	ctx.Logger.Infof("Internal A record %v added", internalDomain)
	return nil
}

func DeleteARecords(ctx *contexts.ClusterContext) error {
	clusterDomain := ctx.Extra.Domain(ctx.Name)
	if err := ctx.DNSProvider.DeleteARecords(clusterDomain); err == nil {
		ctx.Logger.Infof("Cluster apps A record %v deleted", clusterDomain)
	}

	externalDomain := ctx.Extra.ExternalDomain(ctx.Name)
	if err := ctx.DNSProvider.DeleteARecords(externalDomain); err == nil {
		ctx.Logger.Infof("External A record %v deleted", externalDomain)
	}

	internalDomain := ctx.Extra.InternalDomain(ctx.Name)
	if err := ctx.DNSProvider.DeleteARecords(internalDomain); err == nil {
		ctx.Logger.Infof("Internal A record %v deleted", internalDomain)
	}

	return nil
}

func EnsureDnsIPLookup(ctx *contexts.ClusterContext) error {
	externalDomain := ctx.Extra.ExternalDomain(ctx.Name)
	attempt := 0
	for attempt < 120 {
		ips, err := net.LookupIP(externalDomain)
		if len(ips) > 0 && err == nil {
			return nil
		}

		ctx.Logger.Infof("Verifying external DNS %v ... attempt no. %v", externalDomain, attempt)
		time.Sleep(time.Duration(30) * time.Second)
		attempt++
	}

	internalDomain := ctx.Extra.InternalDomain(ctx.Name)
	attempt = 0
	for attempt < 120 {
		ips, err := net.LookupIP(internalDomain)
		if len(ips) > 0 && err == nil {
			return nil
		}

		ctx.Logger.Infof("Verifying internal DNS %v .. attempt no. %v", internalDomain, attempt)
		time.Sleep(time.Duration(30) * time.Second)
		attempt++
	}
	return errors.New("Master DNS failed to propagate in allocated time slot").WithContext(ctx).Err()
}

func ProbeKubeAPI(ctx *contexts.ClusterContext) error {
	/*
		curl --cacert "${CERT_DIR}/pki/ca.crt" \
		  -H "Authorization: Bearer ${KUBE_BEARER_TOKEN}" \
		  ${secure} \
		  --max-time 5 --fail --output /dev/null --silent \
		  "https://${KUBE_MASTER_IP}/api/v1/pods"
	*/
	caCert, err := base64.StdEncoding.DecodeString(ctx.CaCert)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	ctx.DetectApiServerURL()
	url := ctx.ApiServerUrl + "/api"
	mTLSConfig := &tls.Config{}
	certs := x509.NewCertPool()
	certs.AppendCertsFromPEM([]byte(caCert))
	mTLSConfig.RootCAs = certs
	tr := &http.Transport{
		TLSClientConfig: mTLSConfig,
	}

	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", ctx.KubeletToken))
	attempt := 0
	// try for 30 mins
	ctx.Logger.Info("Checking Api")
	for attempt < 40 {
		ctx.Logger.Infof("Attempt %v: probing kubernetes api for cluster %v ...", attempt, ctx.Name)
		_, err := client.Do(req)
		fmt.Print("=")
		if err == nil {
			ctx.Logger.Infof("Successfully connected to kubernetes api for cluster %v", ctx.Name)
			return nil
		}
		attempt++
		time.Sleep(time.Duration(30) * time.Second)
	}
	return errors.Newf("Failed to connect to kubernetes api for cluster %v", ctx.Name).WithContext(ctx).Err()
}

func CheckComponentStatuses(ctx *contexts.ClusterContext) error {
	kubeClient, err := ctx.NewKubeClient()
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	backoff.Retry(func() error {
		resp, err := kubeClient.Client.CoreV1().ComponentStatuses().List(metav1.ListOptions{
			LabelSelector: labels.Everything().String(),
		})
		if err != nil {
			return err
		}
		for _, status := range resp.Items {
			for _, cond := range status.Conditions {
				if cond.Type == apiv1.ComponentHealthy && cond.Status != apiv1.ConditionTrue {
					return errors.New().WithMessagef("Component %v is in condition %v with status %v", status.Name, cond.Type, cond.Status).Err()
				}
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())
	ctx.Logger.Info("Basic componenet status are ok")
	return nil
}

func DeleteNodeApiCall(ctx *contexts.ClusterContext, name string) error {
	kubeClient, err := ctx.NewKubeClient()
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	return kubeClient.Client.CoreV1().Nodes().Delete(name, &metav1.DeleteOptions{})
}

func WaitForReadyNodes(ctx *contexts.ClusterContext, newNode ...int64) error {
	kubeClient, err := ctx.NewKubeClient()
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	var adjust int64 = 0
	if len(newNode) > 0 {
		adjust = newNode[0]
	}
	totalNode := ctx.NodeCount() + adjust
	ctx.Logger.Debug("Number of Nodes = ", totalNode, "adjust = ", adjust)
	attempt := 0
	for attempt < 30 {
		isReady := 0
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"NAME", "LABELS", "STATUS"})

		nodes := &apiv1.NodeList{}
		if kubeClient.Client.CoreV1().RESTClient().Get().Resource("nodes").Do().Into(nodes); err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					isReady++

					row := []string{node.Name, "api.io/hostname=" + node.ObjectMeta.Labels["api.io/hostname"], "Ready"}
					table.Append(row)
				}
			}
		}
		table.SetBorder(true)
		if isReady == int(totalNode) {
			ctx.Logger.Info("All nodes are ready")
			table.Render()
			return nil
		}
		ctx.Logger.Infof("%v nodes ready, waiting...", isReady)
		attempt++
		time.Sleep(time.Duration(60) * time.Second)
	}
	return errors.New("Nodes are not ready after allocated wait time.").WithContext(ctx).Err()
}

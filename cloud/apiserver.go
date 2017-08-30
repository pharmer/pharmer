package cloud

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/appscode/go/errors"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/pharmer/api"
	"github.com/cenkalti/backoff"
	"github.com/olekukonko/tablewriter"
	"github.com/tamalsaha/go-oneliners"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
)

// WARNING:
// Returned KubeClient uses admin bearer token. This should only be used for cluster provisioning operations.
func NewAdminClient(ctx context.Context, cluster *api.Cluster) (clientset.Interface, error) {
	cfg := &rest.Config{
		Host: cluster.ApiServerURL(),
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(CACert(ctx)),
			CertData: cert.EncodeCertPEM(AdminUserCert(ctx)),
			KeyData:  cert.EncodePrivateKeyPEM(AdminUserKey(ctx)),
		},
	}
	return clientset.NewForConfig(cfg)
}

func ProbeKubeAPI(ctx context.Context, cluster *api.Cluster) error {
	/*
		curl --cacert "${CERT_DIR}/pki/ca.crt" \
		  -H "Authorization: Bearer ${KUBE_BEARER_TOKEN}" \
		  ${secure} \
		  --max-time 5 --fail --output /dev/null --silent \
		  "https://${KUBE_MASTER_IP}/api/v1/pods"
	*/
	oneliners.FILE()
	url := cluster.ApiServerURL() + "/api"
	oneliners.FILE()
	mTLSConfig := &tls.Config{}
	certs := x509.NewCertPool()
	oneliners.FILE()
	certs.AppendCertsFromPEM(cert.EncodeCertPEM(CACert(ctx)))
	oneliners.FILE()
	mTLSConfig.RootCAs = certs
	tr := &http.Transport{
		TLSClientConfig: mTLSConfig,
	}
	oneliners.FILE()
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", url, nil)
	oneliners.FILE()
	// req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", cluster.Spec.KubeletToken))
	attempt := 0
	// try for 30 mins
	oneliners.FILE()
	Logger(ctx).Info("Checking Api")
	for attempt < 40 {
		Logger(ctx).Infof("Attempt %v: probing kubernetes api for cluster %v ...", attempt, cluster.Name)
		_, err := client.Do(req)
		fmt.Print("=")
		if err == nil {
			Logger(ctx).Infof("Successfully connected to kubernetes api for cluster %v", cluster.Name)
			return nil
		}
		attempt++
		time.Sleep(time.Duration(30) * time.Second)
	}
	return errors.Newf("Failed to connect to kubernetes api for cluster %v", cluster.Name).WithContext(ctx).Err()
}

func CheckComponentStatuses(ctx context.Context, cluster *api.Cluster) error {
	kubeClient, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	backoff.Retry(func() error {
		resp, err := kubeClient.CoreV1().ComponentStatuses().List(metav1.ListOptions{
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
	Logger(ctx).Info("Basic componenet status are ok")
	return nil
}

func DeleteNodeApiCall(ctx context.Context, cluster *api.Cluster, name string) error {
	kubeClient, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	return kubeClient.CoreV1().Nodes().Delete(name, &metav1.DeleteOptions{})
}

func WaitForReadyNodes(ctx context.Context, cluster *api.Cluster, newNode ...int64) error {
	kubeClient, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	var adjust int64 = 0
	if len(newNode) > 0 {
		adjust = newNode[0]
	}
	totalNode := cluster.NodeCount() + adjust
	Logger(ctx).Debug("Number of Nodes = ", totalNode, "adjust = ", adjust)
	attempt := 0
	for attempt < 30 {
		isReady := 0
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"NAME", "LABELS", "STATUS"})

		nodes := &apiv1.NodeList{}
		if kubeClient.CoreV1().RESTClient().Get().Resource("nodes").Do().Into(nodes); err != nil {
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
			Logger(ctx).Info("All nodes are ready")
			table.Render()
			return nil
		}
		Logger(ctx).Infof("%v nodes ready, waiting...", isReady)
		attempt++
		time.Sleep(time.Duration(60) * time.Second)
	}
	return errors.New("Nodes are not ready after allocated wait time.").WithContext(ctx).Err()
}

var restrictedNamespaces []string = []string{"appscode", "kube-system"}

func hasNoUserApps(ctx context.Context, clusterName string, client clientset.Interface) (bool, error) {
	pods, err := client.CoreV1().Pods(apiv1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		// If we can't connect to kube apiserver, then delete cluster.
		// Cluster probably failed to create.
		return true, nil
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" && !stringz.Contains(restrictedNamespaces, pod.Namespace) {
			return false, nil
		}
	}
	return true, nil
}

func deleteLoadBalancers(client clientset.Interface) error {
	// Delete services with type = LoadBalancer
	backoff.Retry(func() error {
		svcs, err := client.CoreV1().Services("").List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, svc := range svcs.Items {
			if svc.Spec.Type == apiv1.ServiceTypeLoadBalancer {
				trueValue := true
				err = client.CoreV1().Services(svc.Namespace).Delete(svc.Name, &metav1.DeleteOptions{OrphanDependents: &trueValue})
				if err != nil {
					return err
				}
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	return nil
}

func deleteDyanamicVolumes(client clientset.Interface) error {
	backoff.Retry(func() error {
		pvcs, err := client.CoreV1().PersistentVolumeClaims("").List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, pvc := range pvcs.Items {
			if pvc.Status.Phase == apiv1.ClaimBound {
				for k, v := range pvc.Annotations {
					if (k == "volume.alpha.kubernetes.io/storage-class" ||
						k == "volume.beta.kubernetes.io/storage-class") && v != "" {
						trueValue := true
						err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{OrphanDependents: &trueValue})
						if err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())
	return nil
}

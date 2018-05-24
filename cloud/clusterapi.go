package cloud

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"text/template"
	"time"

	api "github.com/pharmer/pharmer/apis/v1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

type ClusterApi struct {
	ctx     context.Context
	cluster *api.Cluster

	token     string
	kc        kubernetes.Interface
	client    v1alpha1.ClusterV1alpha1Interface
	clientSet clientset.Interface
}

type ApiServerTemplate struct {
	ClusterName            string
	Token                  string
	APIServerImage         string
	ControllerManagerImage string
	MachineControllerImage string
	CABundle               string
	TLSCrt                 string
	TLSKey                 string
	Provider               string
	MasterCount            int
}

var apiServerImage = "pharmer/cluster-apiserver:0.0.3"
var controllerManagerImage = "pharmer/cluster-controller-manager:0.0.3"
var machineControllerImage = "pharmer/machine-controller:clusterApi"

const (
	BasePath = ".pharmer/config.d"
)

func NewClusterApi(ctx context.Context, cluster *api.Cluster, kc kubernetes.Interface) (*ClusterApi, error) {
	c, err := NewClusterApiClient(ctx, cluster)
	if err != nil {
		return nil, err
	}
	var token string
	if token, err = GetExistingKubeadmToken(kc, kubeadmconsts.DefaultTokenDuration); err != nil {
		return nil, err
	}

	return &ClusterApi{ctx: ctx, cluster: cluster, kc: kc, token: token, client: c.ClusterV1alpha1(), clientSet: c}, nil
}

func (ca *ClusterApi) Apply() error {
	Logger(ca.ctx).Infoln("using cluster locally")
	c2, err := GetAdminConfig(ca.ctx, ca.cluster)
	if err != nil {
		Logger(ca.ctx).Infoln("error on using cluster", err)
		return err
	}
	opt := options.NewClusterUseConfig()
	opt.ClusterName = ca.cluster.Name
	UseCluster(ca.ctx, opt, c2)

	if err := waitForServiceAccount(ca.ctx, ca.kc); err != nil {
		return err
	}
	Logger(ca.ctx).Infof("Deploying the addon apiserver and controller manager...")
	if err := ca.CreateMachineController(); err != nil {
		return fmt.Errorf("can't create machine controller: %v", err)
	}

	if err := waitForClusterResourceReady(ca.ctx, ca.clientSet); err != nil {
		return err
	}
	if _, err := ca.client.Clusters(core.NamespaceDefault).Create(ca.cluster.Spec.ClusterAPI); err != nil {
		return err
	}

	if _, err := ca.client.Clusters(core.NamespaceDefault).UpdateStatus(ca.cluster.Spec.ClusterAPI); err != nil {
		return err
	}

	Logger(ca.ctx).Infof("Adding master machines...")
	for _, master := range ca.cluster.Spec.Masters {
		if _, err := ca.client.Machines(core.NamespaceDefault).Create(master); err != nil {
			return err
		}
	}

	return nil
}

func (ca *ClusterApi) CreateMachineController() error {
	Logger(ca.ctx).Infoln("creating pharmer secret")
	if err := ca.CreatePharmerSecret(); err != nil {
		return err
	}

	Logger(ca.ctx).Infoln("creating cluster api rolebinding")
	if err := ca.CreateExtApiServerRoleBinding(); err != nil {
		return err
	}

	Logger(ca.ctx).Infoln("creating apiserver and controller")
	if err := ca.CreateApiServerAndController(); err != nil {
		return err
	}
	return nil
}

func (ca *ClusterApi) CreatePharmerSecret() error {
	providerConfig := ca.cluster.ProviderConfig()

	cred, err := Store(ca.ctx).Credentials().Get(ca.cluster.Spec.CredentialName)
	if err != nil {
		return err
	}
	credData, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	if err = CreateSecret(ca.kc, "pharmer-cred", map[string][]byte{
		fmt.Sprintf("%v.json", ca.cluster.Spec.CredentialName): credData,
	}); err != nil {
		return err
	}

	cluster, err := json.MarshalIndent(ca.cluster, "", "  ")
	if err != nil {
		return err
	}
	if err = CreateSecret(ca.kc, "pharmer-cluster", map[string][]byte{
		fmt.Sprintf("%v.json", ca.cluster.Name): cluster,
	}); err != nil {
		return err
	}

	publicKey, privateKey, err := Store(ca.ctx).SSHKeys(ca.cluster.Name).Get(ca.cluster.ProviderConfig().SSHKeyName)
	if err != nil {
		return err
	}
	if err = CreateSecret(ca.kc, "pharmer-ssh", map[string][]byte{
		fmt.Sprintf("id_%v", providerConfig.SSHKeyName):     privateKey,
		fmt.Sprintf("id_%v.pub", providerConfig.SSHKeyName): publicKey,
	}); err != nil {
		return err
	}

	if err = CreateSecret(ca.kc, "pharmer-certificate", map[string][]byte{
		"ca.crt":             cert.EncodeCertPEM(CACert(ca.ctx)),
		"ca.key":             cert.EncodePrivateKeyPEM(CAKey(ca.ctx)),
		"front-proxy-ca.crt": cert.EncodeCertPEM(FrontProxyCACert(ca.ctx)),
		"front-proxy-ca.key": cert.EncodePrivateKeyPEM(FrontProxyCAKey(ca.ctx)),
		"sa.crt":             cert.EncodeCertPEM(SaCert(ca.ctx)),
		"sa.key":             cert.EncodePrivateKeyPEM(SaKey(ca.ctx)),
	}); err != nil {
		return err
	}

	if err = CreateSecret(ca.kc, "pharmer-etcd", map[string][]byte{
		"ca.crt": cert.EncodeCertPEM(EtcdCaCert(ca.ctx)),
		"ca.key": cert.EncodePrivateKeyPEM(EtcdCaKey(ca.ctx)),
	}); err != nil {
		return err
	}

	return nil
}

func (ca *ClusterApi) CreateExtApiServerRoleBinding() error {
	rolebinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machine-controller",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "extension-apiserver-authentication-reader",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: "default",
			},
		},
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := ca.kc.RbacV1().RoleBindings(metav1.NamespaceSystem).Create(rolebinding)
		return err == nil, nil
	})
}

func (ca *ClusterApi) CreateApiServerAndController() error {
	tmpl, err := template.New("config").Parse(ClusterAPIDeployConfigTemplate)
	if err != nil {
		return err
	}
	if ca.ctx, err = LoadApiserverCertificate(ca.ctx, ca.cluster); err != nil {
		return err
	}
	masterNG, err := FindMasterMachines(ca.cluster)
	if err != nil {
		return err
	}

	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, ApiServerTemplate{
		ClusterName:            ca.cluster.Name,
		Token:                  ca.token,
		APIServerImage:         apiServerImage,
		ControllerManagerImage: controllerManagerImage,
		MachineControllerImage: machineControllerImage,
		CABundle:               base64.StdEncoding.EncodeToString(cert.EncodeCertPEM(ApiServerCaCert(ca.ctx))),
		TLSCrt:                 base64.StdEncoding.EncodeToString(cert.EncodeCertPEM(ApiServerCert(ca.ctx))),
		TLSKey:                 base64.StdEncoding.EncodeToString(cert.EncodePrivateKeyPEM(ApiServerKey(ca.ctx))),
		Provider:               ca.cluster.ProviderConfig().CloudProvider,
		MasterCount:            len(masterNG),
	})
	if err != nil {
		return err
	}

	maxTries := 5
	for tries := 0; tries < maxTries; tries++ {
		err = deployConfig(tmplBuf.Bytes())
		if err == nil {
			return nil
		} else {
			if tries < maxTries-1 {
				//glog.Info("Error scheduling machine controller. Will retry...\n", err)
				time.Sleep(3 * time.Second)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("couldn't start machine controller: %v\n", err)
	} else {
		return nil
	}

}

func deployConfig(manifest []byte) error {
	fmt.Println(string(manifest))
	cmd := exec.Command("kubectl", "create", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		stdin.Write(manifest)
	}()

	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	} else {
		return fmt.Errorf("couldn't create pod: %v, output: %s", err, string(out))
	}
}

const ClusterAPIDeployConfigTemplate = `
apiVersion: apiregistration.k8s.io/v1beta1
kind: APIService
metadata:
  name: v1alpha1.cluster.k8s.io
  labels:
    api: clusterapi
    apiserver: "true"
spec:
  version: v1alpha1
  group: cluster.k8s.io
  groupPriorityMinimum: 2000
  priority: 200
  service:
    name: clusterapi
    namespace: default
  versionPriority: 10
  caBundle: {{ .CABundle }}
---
apiVersion: v1
kind: Service
metadata:
  name: clusterapi
  namespace: default
  labels:
    api: clusterapi
    apiserver: "true"
spec:
  ports:
  - port: 443
    protocol: TCP
    targetPort: 443
  selector:
    api: clusterapi
    apiserver: "true"
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: clusterapi
  namespace: default
  labels:
    api: clusterapi
    apiserver: "true"
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ''
spec:
  template:
    metadata:
      labels:
        api: clusterapi
        apiserver: "true"
    spec:
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
      - key: CriticalAddonsOnly
        operator: Exists
      - effect: NoExecute
        key: node.alpha.kubernetes.io/notReady
        operator: Exists
      - effect: NoExecute
        key: node.alpha.kubernetes.io/unreachable
        operator: Exists
      containers:
      - name: apiserver
        image: {{ .APIServerImage }}
        imagePullPolicy: Always
        volumeMounts:
        - name: cluster-apiserver-certs
          mountPath: /apiserver.local.config/certificates
          readOnly: true
        - name: config
          mountPath: /etc/kubernetes
        - name: certs
          mountPath: /etc/ssl/certs
        command:
        - "./apiserver"
        args:
        - "--etcd-servers=http://etcd-clusterapi-svc:2379"
        - "--tls-cert-file=/apiserver.local.config/certificates/tls.crt"
        - "--tls-private-key-file=/apiserver.local.config/certificates/tls.key"
        - "--audit-log-path=-"
        - "--audit-log-maxage=0"
        - "--audit-log-maxbackup=0"
        - "--authorization-kubeconfig=/etc/kubernetes/admin.conf"
        - "--authentication-kubeconfig=/etc/kubernetes/admin.conf"
        - "--kubeconfig=/etc/kubernetes/admin.conf"
        resources:
          requests:
            cpu: 100m
            memory: 20Mi
          limits:
            cpu: 100m
            memory: 30Mi
      - name: controller-manager
        image: {{ .ControllerManagerImage }}
        imagePullPolicy: Always
        volumeMounts:
          - name: config
            mountPath: /etc/kubernetes
          - name: certs
            mountPath: /etc/ssl/certs
        command:
        - "./controller-manager"
        args:
        - --kubeconfig=/etc/kubernetes/admin.conf
        resources:
          requests:
            cpu: 100m
            memory: 20Mi
          limits:
            cpu: 100m
            memory: 30Mi
      - name: machine-controller
        image: {{ .MachineControllerImage }}
        imagePullPolicy: Always
        volumeMounts:
          - name: config
            mountPath: /etc/kubernetes
          - name: certs
            mountPath: /etc/ssl/certs
          - name: sshkeys
            mountPath: /root/.pharmer/store.d/clusters/{{ .ClusterName }}/ssh
          - name: certificates
            mountPath: /root/.pharmer/store.d/clusters/{{ .ClusterName }}/pki
          - name: etcd-cert
            mountPath: /root/.pharmer/store.d/clusters/{{ .ClusterName }}/pki/etcd
          - name: cluster
            mountPath: /root/.pharmer/store.d/clusters
          - name: credential
            mountPath: /root/.pharmer/store.d/credentials
        args:
        - controller
        - --kubeconfig=/etc/kubernetes/admin.conf
        - --provider={{ .Provider }}
        - --v=5
        resources:
          requests:
            cpu: 100m
            memory: 20Mi
          limits:
            cpu: 100m
            memory: 30Mi
      volumes:
      - name: cluster-apiserver-certs
        secret:
          secretName: cluster-apiserver-certs
      - name: config
        hostPath:
          path: /etc/kubernetes
      - name: certs
        hostPath:
          path: /etc/ssl/certs
      - name: sshkeys
        secret:
          secretName: pharmer-ssh
          defaultMode: 256
      - name: certificates
        secret:
          secretName: pharmer-certificate
          defaultMode: 256
      - name: etcd-cert
        secret:
          secretName: pharmer-etcd
          defaultMode: 256
      - name: cluster
        secret:
          secretName: pharmer-cluster
          defaultMode: 256
      - name: credential
        secret:
          secretName: pharmer-cred
          defaultMode: 256

---
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: etcd-clusterapi
  namespace: default
spec:
  serviceName: "etcd"
  replicas: 1
  template:
    metadata:
      labels:
        app: etcd
    spec:
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
      - key: CriticalAddonsOnly
        operator: Exists
      - effect: NoExecute
        key: node.alpha.kubernetes.io/notReady
        operator: Exists
      - effect: NoExecute
        key: node.alpha.kubernetes.io/unreachable
        operator: Exists
      volumes:
      - hostPath:
          path: /var/lib/etcd2
          type: DirectoryOrCreate
        name: etcd-data-dir
      terminationGracePeriodSeconds: 10
      containers:
      - name: etcd
        image: quay.io/coreos/etcd:latest
        imagePullPolicy: Always
        resources:
          requests:
            cpu: 100m
            memory: 20Mi
          limits:
            cpu: 100m
            memory: 30Mi
        env:
        - name: ETCD_DATA_DIR
          value: /etcd-data-dir
        command:
        - /usr/local/bin/etcd
        - --listen-client-urls
        - http://0.0.0.0:2379
        - --advertise-client-urls
        - http://localhost:2379
        ports:
        - containerPort: 2379
        volumeMounts:
        - name: etcd-data-dir
          mountPath: /etcd-data-dir
        readinessProbe:
          httpGet:
            port: 2379
            path: /health
          failureThreshold: 3
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 10
        livenessProbe:
          httpGet:
            port: 2379
            path: /health
          failureThreshold: 8
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 15
---
apiVersion: v1
kind: Service
metadata:
  name: etcd-clusterapi-svc
  namespace: default
  labels:
    app: etcd
spec:
  ports:
  - port: 2379
    name: etcd
    targetPort: 2379
  selector:
    app: etcd
---
apiVersion: v1
kind: Secret
type: kubernetes.io/tls
metadata:
  name: cluster-apiserver-certs
  namespace: default
  labels:
    api: clusterapi
    apiserver: "true"
data:
  tls.crt: {{ .TLSCrt }}
  tls.key: {{ .TLSKey }}
`

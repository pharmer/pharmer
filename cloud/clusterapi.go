package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/phases"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterApi struct {
	ctx     context.Context
	cluster *api.Cluster

	namespace string
	token     string
	kc        kubernetes.Interface
	client    client.Client

	bootstrapClient clusterclient.Client
}

type ApiServerTemplate struct {
	ClusterName         string
	Provider            string
	ControllerNamespace string
	ControllerImage     string
}

var machineControllerImage = "pharmer/machine-controller:clusterapi"

const (
	BasePath = ".pharmer/config.d"
)

func NewClusterApi(ctx context.Context, cluster *api.Cluster, namespace string, kc kubernetes.Interface) (*ClusterApi, error) {
	var token string
	var err error
	if token, err = GetExistingKubeadmToken(kc, kubeadmconsts.DefaultTokenDuration); err != nil {
		return nil, err
	}

	bc, err := GetBooststrapClient(ctx, cluster)
	if err != nil {
		return nil, err
	}

	return &ClusterApi{ctx: ctx, cluster: cluster, namespace: namespace, kc: kc, token: token, bootstrapClient: bc}, nil
}

func (ca *ClusterApi) Apply() error {

	Logger(ca.ctx).Infof("Deploying the addon apiserver and controller manager...")
	if err := ca.CreateMachineController(); err != nil {
		return fmt.Errorf("can't create machine controller: %v", err)
	}

	if err := phases.ApplyCluster(ca.bootstrapClient, ca.cluster.Spec.ClusterAPI); err != nil {
		return err
	}
	namespace := ca.cluster.Spec.ClusterAPI.Namespace
	if namespace == "" {
		namespace = ca.bootstrapClient.GetContextNamespace()
	}
	machines, err := Store(ca.ctx).Machine(ca.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	masterMachine, err := api.GetMasterMachine(machines)
	if err != nil {
		return err
	}

	masterMachine.Annotations = make(map[string]string)
	masterMachine.Annotations[InstanceStatusAnnotationKey] = ""

	Logger(ca.ctx).Infof("Adding master machines...")
	if err := phases.ApplyMachines(ca.bootstrapClient, namespace, []*clusterv1.Machine{masterMachine}); err != nil {
		return err
	}

	return nil
}

func (ca *ClusterApi) CreateMachineController() error {
	Logger(ca.ctx).Infoln("creating pharmer secret")
	if err := ca.CreatePharmerSecret(); err != nil {
		return err
	}

	Logger(ca.ctx).Infoln("creating apiserver and controller")
	if err := ca.CreateApiServerAndController(); err != nil {
		return err
	}
	return nil
}

func (ca *ClusterApi) CreatePharmerSecret() error {
	providerConfig := ca.cluster.ClusterConfig()

	cred, err := Store(ca.ctx).Credentials().Get(ca.cluster.ClusterConfig().CredentialName)
	if err != nil {
		return err
	}
	credData, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}

	err = CreateNamespace(ca.kc, ca.namespace)
	fmt.Println(err)

	if err = CreateSecret(ca.kc, "pharmer-cred", ca.namespace, map[string][]byte{
		fmt.Sprintf("%v.json", ca.cluster.ClusterConfig().CredentialName): credData,
	}); err != nil {
		return err
	}

	cluster, err := json.MarshalIndent(ca.cluster, "", "  ")
	if err != nil {
		return err
	}
	if err = CreateSecret(ca.kc, "pharmer-cluster", ca.namespace, map[string][]byte{
		fmt.Sprintf("%v.json", ca.cluster.Name): cluster,
	}); err != nil {
		return err
	}

	publicKey, privateKey, err := Store(ca.ctx).SSHKeys(ca.cluster.Name).Get(ca.cluster.ClusterConfig().Cloud.SSHKeyName)
	if err != nil {
		return err
	}
	if err = CreateSecret(ca.kc, "pharmer-ssh", ca.namespace, map[string][]byte{
		fmt.Sprintf("id_%v", providerConfig.Cloud.SSHKeyName):     privateKey,
		fmt.Sprintf("id_%v.pub", providerConfig.Cloud.SSHKeyName): publicKey,
	}); err != nil {
		return err
	}

	if err = CreateSecret(ca.kc, "pharmer-certificate", ca.namespace, map[string][]byte{
		"ca.crt":             cert.EncodeCertPEM(CACert(ca.ctx)),
		"ca.key":             cert.EncodePrivateKeyPEM(CAKey(ca.ctx)),
		"front-proxy-ca.crt": cert.EncodeCertPEM(FrontProxyCACert(ca.ctx)),
		"front-proxy-ca.key": cert.EncodePrivateKeyPEM(FrontProxyCAKey(ca.ctx)),
		//"sa.crt":             cert.EncodeCertPEM(SaCert(ca.ctx)),
		//"sa.key":             cert.EncodePrivateKeyPEM(SaKey(ca.ctx)),
	}); err != nil {
		return err
	}

	/*if err = CreateSecret(ca.kc, "pharmer-etcd", map[string][]byte{
		"ca.crt": cert.EncodeCertPEM(EtcdCaCert(ca.ctx)),
		"ca.key": cert.EncodePrivateKeyPEM(EtcdCaKey(ca.ctx)),
	}); err != nil {
		return err
	}*/

	return nil
}

func (ca *ClusterApi) CreateApiServerAndController() error {
	tmpl, err := template.New("config").Parse(ClusterAPIDeployConfigTemplate)
	if err != nil {
		return err
	}
	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, ApiServerTemplate{
		ClusterName:         ca.cluster.Name,
		Provider:            ca.cluster.ClusterConfig().Cloud.CloudProvider,
		ControllerNamespace: ca.namespace,
		ControllerImage:     machineControllerImage,
	})
	if err != nil {
		return err
	}

	return ca.bootstrapClient.Apply(tmplBuf.String())

}

const ClusterAPIDeployConfigTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: {{ .ControllerNamespace }}
---
apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
  name: do-provider-controller-manager-service
  namespace: {{ .ControllerNamespace }}
spec:
  ports:
  - port: 443
  selector:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
  name: do-provider-controller-manager
  namespace: {{ .ControllerNamespace }}
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
      controller-tools.k8s.io: "1.0"
  serviceName: do-provider-controller-manager-service
  template:
    metadata:
      labels:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
    spec:
      nodeSelector:
        node-role.kubernetes.io/master: ""
      containers:
      - args:
        - controller
        - --provider={{ .Provider }}
        - --kubeconfig=/etc/kubernetes/admin.conf 
        env:
        image: {{ .ControllerImage }}
        name: manager
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
        volumeMounts:
        - mountPath: /etc/kubernetes
          name: config
        - mountPath: /etc/ssl/certs
          name: certs
        - name: sshkeys
          mountPath: /root/.pharmer/store.d/clusters/{{ .ClusterName }}/ssh
        - name: certificates
          mountPath: /root/.pharmer/store.d/clusters/{{ .ClusterName }}/pki
        - name: cluster
          mountPath: /root/.pharmer/store.d/clusters
        - name: credential
          mountPath: /root/.pharmer/store.d/credentials
      terminationGracePeriodSeconds: 10
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
          path: /etc/kubernetes
        name: config
      - hostPath:
          path: /etc/ssl/certs
        name: certs
      - name: sshkeys
        secret:
          secretName: pharmer-ssh
          defaultMode: 256
      - name: certificates
        secret:
          secretName: pharmer-certificate
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
apiVersion: v1
kind: Namespace
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: cluster-api-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: cluster-api-manager-role
rules:
- apiGroups:
  - cluster.k8s.io
  resources:
  - clusters
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - cluster.k8s.io
  resources:
  - machines
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - cluster.k8s.io
  resources:
  - machinedeployments
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - cluster.k8s.io
  resources:
  - machinesets
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - cluster.k8s.io
  resources:
  - machines
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - ""
  resources:
  - nodes
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - cluster.k8s.io
  resources:
  - machines
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  creationTimestamp: null
  name: cluster-api-manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-api-manager-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: cluster-api-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
  name: cluster-api-controller-manager-service
  namespace: cluster-api-system
spec:
  ports:
  - port: 443
  selector:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
  name: cluster-api-controller-manager
  namespace: cluster-api-system
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
      controller-tools.k8s.io: "1.0"
  serviceName: cluster-api-controller-manager-service
  template:
    metadata:
      labels:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
    spec:
      containers:
      - command:
        - /manager
        image: gcr.io/k8s-cluster-api/cluster-api-controller:latest
        name: manager
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
      terminationGracePeriodSeconds: 10
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
`

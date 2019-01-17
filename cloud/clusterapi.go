package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	api "github.com/pharmer/pharmer/apis/v1beta1"
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
	/*for _, master := range ca.cluster.Spec.Masters {
		if _, err := ca.client.Machines(core.NamespaceDefault).Create(master); err != nil {
			return err
		}
	}
	*/
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
	providerConfig := ca.cluster.ClusterConfig()

	cred, err := Store(ca.ctx).Credentials().Get(ca.cluster.ClusterConfig().CredentialName)
	if err != nil {
		return err
	}
	credData, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	if err = CreateSecret(ca.kc, "pharmer-cred", map[string][]byte{
		fmt.Sprintf("%v.json", ca.cluster.ClusterConfig().CredentialName): credData,
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

	publicKey, privateKey, err := Store(ca.ctx).SSHKeys(ca.cluster.Name).Get(ca.cluster.ClusterConfig().Cloud.SSHKeyName)
	if err != nil {
		return err
	}
	if err = CreateSecret(ca.kc, "pharmer-ssh", map[string][]byte{
		fmt.Sprintf("id_%v", providerConfig.Cloud.SSHKeyName):     privateKey,
		fmt.Sprintf("id_%v.pub", providerConfig.Cloud.SSHKeyName): publicKey,
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
	return nil
	/*tmpl, err := template.New("config").Parse(ClusterAPIDeployConfigTemplate)
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
		Provider:               ca.cluster.ClusterConfig().CloudProvider,
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
	*/
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

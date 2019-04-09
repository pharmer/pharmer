package gce

import (
	"bytes"
	"encoding/base64"
	"strings"
	"text/template"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"gopkg.in/yaml.v2"
)

func (conn *cloudConnector) getControllerManager() (string, error) {
	config := conn.cluster.ClusterConfig()
	cred, err := Store(conn.ctx).Owner(conn.owner).Credentials().Get(config.CredentialName)
	if err != nil {
		return "", err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}

	machineSetup, err := geteMachineSetupConfig(config)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("controller-manager-config").Parse(ControllerManager)
	if err != nil {
		return "", err
	}

	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, controllerManagerConfig{
		MachineConfig:   machineSetup,
		ServiceAccount:  base64.StdEncoding.EncodeToString([]byte(typed.ServiceAccount())),
		SSHPrivateKey:   base64.StdEncoding.EncodeToString((SSHKey(conn.ctx).PrivateKey)),
		SSHPublicKey:    base64.StdEncoding.EncodeToString((SSHKey(conn.ctx).PublicKey)),
		SSHUser:         base64.StdEncoding.EncodeToString([]byte("clusterapi")),
		ControllerImage: MachineControllerImage,
	})
	if err != nil {
		return "", err
	}

	return tmplBuf.String(), nil
}

func geteMachineSetupConfig(config *api.ClusterConfig) (string, error) {
	tmpl, err := template.New("machine-config").Parse(machineSetupConfig)
	if err != nil {
		return "", err
	}
	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, setupConfig{
		OS:                  config.Cloud.InstanceImage,
		OSFamily:            config.Cloud.OS,
		KubeletVersion:      strings.Trim(config.KubernetesVersion, "vV"),
		ControlPlaneVersion: strings.Trim(config.KubernetesVersion, "vV"),
	})
	if err != nil {
		return "", err
	}

	data, err := yaml.Marshal(tmplBuf.String())

	if err != nil {
		return "", err
	}

	return string(data), nil
}

type controllerManagerConfig struct {
	MachineConfig   string
	ServiceAccount  string
	SSHPrivateKey   string
	SSHPublicKey    string
	SSHUser         string
	ControllerImage string
}

const ControllerManager = `
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: gcp-provider-system
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: gceclusterproviderspecs.gceproviderconfig.k8s.io
spec:
  group: gceproviderconfig.k8s.io
  names:
    kind: GCEClusterProviderSpec
    plural: gceclusterproviderspecs
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        project:
          type: string
      required:
      - project
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: gcemachineproviderspecs.gceproviderconfig.k8s.io
spec:
  group: gceproviderconfig.k8s.io
  names:
    kind: GCEMachineProviderSpec
    plural: gcemachineproviderspecs
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        disks:
          items:
            properties:
              initializeParams:
                properties:
                  diskSizeGb:
                    format: int64
                    type: integer
                  diskType:
                    type: string
                required:
                - diskSizeGb
                - diskType
                type: object
            required:
            - initializeParams
            type: object
          type: array
        kind:
          type: string
        machineType:
          type: string
        metadata:
          type: object
        os:
          type: string
        roles:
          items:
            type: string
          type: array
        zone:
          type: string
      required:
      - zone
      - machineType
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gcp-provider-manager-role
rules:
- apiGroups:
  - gceproviderconfig.k8s.io
  resources:
  - gceclusterproviderconfigs
  - gcemachineproviderconfigs
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
  - clusters
  - machines
  - machines/status
  - machinedeployments
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
  - ""
  resources:
  - nodes
  - events
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
  name: gcp-provider-manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gcp-provider-manager-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: gcp-provider-system
---
apiVersion: v1
data:
  machine_setup_configs.yaml: {{ .MachineConfig }}
kind: ConfigMap
metadata:
  name: gcp-provider-machine-setup
  namespace: gcp-provider-system
---
apiVersion: v1
data:
  service-account.json: {{ .ServiceAccount }}
kind: Secret
metadata:
  name: gcp-provider-machine-controller-credential
  namespace: gcp-provider-system
type: Opaque
---
apiVersion: v1
data:
  private: {{ .SSHPrivateKey }}
  public: {{ .SSHPublicKey }}
  user: {{ .SSHUser }}
kind: Secret
metadata:
  name: gcp-provider-machine-controller-sshkeys
  namespace: gcp-provider-system
type: Opaque
---
apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
  name: gcp-provider-controller-manager-service
  namespace: gcp-provider-system
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
  name: gcp-provider-controller-manager
  namespace: gcp-provider-system
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
      controller-tools.k8s.io: "1.0"
  serviceName: gcp-provider-controller-manager-service
  template:
    metadata:
      labels:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
    spec:
      containers:
      - args:
        - -logtostderr=true
        - -stderrthreshold=INFO
        command:
        - /manager
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /etc/credentials/service-account.json
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
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
        - mountPath: /etc/credentials
          name: credentials
        - mountPath: /etc/sshkeys
          name: sshkeys
        - mountPath: /etc/machinesetup
          name: machine-setup
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
          defaultMode: 256
          secretName: gcp-provider-machine-controller-sshkeys
      - name: credentials
        secret:
          secretName: gcp-provider-machine-controller-credential
      - configMap:
          name: gcp-provider-machine-setup
        name: machine-setup
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: cluster-api-system
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: clusters.cluster.k8s.io
spec:
  group: cluster.k8s.io
  names:
    kind: Cluster
    plural: clusters
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            clusterNetwork:
              properties:
                pods:
                  properties:
                    cidrBlocks:
                      items:
                        type: string
                      type: array
                  required:
                  - cidrBlocks
                  type: object
                serviceDomain:
                  type: string
                services:
                  properties:
                    cidrBlocks:
                      items:
                        type: string
                      type: array
                  required:
                  - cidrBlocks
                  type: object
              required:
              - services
              - pods
              - serviceDomain
              type: object
            providerSpec:
              properties:
                value:
                  type: object
                valueFrom:
                  properties:
                    machineClass:
                      properties:
                        provider:
                          type: string
                      type: object
                  type: object
              type: object
          required:
          - clusterNetwork
          type: object
        status:
          properties:
            apiEndpoints:
              items:
                properties:
                  host:
                    type: string
                  port:
                    format: int64
                    type: integer
                required:
                - host
                - port
                type: object
              type: array
            errorMessage:
              type: string
            errorReason:
              type: string
            providerStatus:
              type: object
          type: object
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: machineclasses.cluster.k8s.io
spec:
  group: cluster.k8s.io
  names:
    kind: MachineClass
    plural: machineclasses
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        providerSpec:
          type: object
      required:
      - providerSpec
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: machinedeployments.cluster.k8s.io
spec:
  group: cluster.k8s.io
  names:
    kind: MachineDeployment
    plural: machinedeployments
  scope: Namespaced
  subresources:
    scale:
      labelSelectorPath: .status.labelSelector
      specReplicasPath: .spec.replicas
      statusReplicasPath: .status.replicas
    status: {}
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            minReadySeconds:
              format: int32
              type: integer
            paused:
              type: boolean
            progressDeadlineSeconds:
              format: int32
              type: integer
            replicas:
              format: int32
              type: integer
            revisionHistoryLimit:
              format: int32
              type: integer
            selector:
              type: object
            strategy:
              properties:
                rollingUpdate:
                  properties:
                    maxSurge: {}
                    maxUnavailable: {}
                  type: object
                type:
                  type: string
              type: object
            template:
              properties:
                metadata:
                  type: object
                spec:
                  properties:
                    configSource:
                      type: object
                    metadata:
                      type: object
                    providerSpec:
                      properties:
                        value:
                          type: object
                        valueFrom:
                          properties:
                            machineClass:
                              properties:
                                provider:
                                  type: string
                              type: object
                          type: object
                      type: object
                    taints:
                      items:
                        type: object
                      type: array
                    versions:
                      properties:
                        controlPlane:
                          type: string
                        kubelet:
                          type: string
                      required:
                      - kubelet
                      type: object
                  required:
                  - providerSpec
                  type: object
              type: object
          required:
          - selector
          - template
          type: object
        status:
          properties:
            availableReplicas:
              format: int32
              type: integer
            observedGeneration:
              format: int64
              type: integer
            readyReplicas:
              format: int32
              type: integer
            replicas:
              format: int32
              type: integer
            unavailableReplicas:
              format: int32
              type: integer
            updatedReplicas:
              format: int32
              type: integer
          type: object
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: machines.cluster.k8s.io
spec:
  group: cluster.k8s.io
  names:
    kind: Machine
    plural: machines
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            configSource:
              type: object
            metadata:
              type: object
            providerSpec:
              properties:
                value:
                  type: object
                valueFrom:
                  properties:
                    machineClass:
                      properties:
                        provider:
                          type: string
                      type: object
                  type: object
              type: object
            taints:
              items:
                type: object
              type: array
            versions:
              properties:
                controlPlane:
                  type: string
                kubelet:
                  type: string
              required:
              - kubelet
              type: object
          required:
          - providerSpec
          type: object
        status:
          properties:
            addresses:
              items:
                type: object
              type: array
            conditions:
              items:
                type: object
              type: array
            errorMessage:
              type: string
            errorReason:
              type: string
            lastOperation:
              properties:
                description:
                  type: string
                lastUpdated:
                  format: date-time
                  type: string
                state:
                  type: string
                type:
                  type: string
              type: object
            lastUpdated:
              format: date-time
              type: string
            nodeRef:
              type: object
            phase:
              type: string
            providerStatus:
              type: object
            versions:
              properties:
                controlPlane:
                  type: string
                kubelet:
                  type: string
              required:
              - kubelet
              type: object
          type: object
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  labels:
    controller-tools.k8s.io: "1.0"
  name: machinesets.cluster.k8s.io
spec:
  group: cluster.k8s.io
  names:
    kind: MachineSet
    plural: machinesets
  scope: Namespaced
  subresources:
    scale:
      labelSelectorPath: .status.labelSelector
      specReplicasPath: .spec.replicas
      statusReplicasPath: .status.replicas
    status: {}
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            minReadySeconds:
              format: int32
              type: integer
            replicas:
              format: int32
              type: integer
            selector:
              type: object
            template:
              properties:
                metadata:
                  type: object
                spec:
                  properties:
                    configSource:
                      type: object
                    metadata:
                      type: object
                    providerSpec:
                      properties:
                        value:
                          type: object
                        valueFrom:
                          properties:
                            machineClass:
                              properties:
                                provider:
                                  type: string
                              type: object
                          type: object
                      type: object
                    taints:
                      items:
                        type: object
                      type: array
                    versions:
                      properties:
                        controlPlane:
                          type: string
                        kubelet:
                          type: string
                      required:
                      - kubelet
                      type: object
                  required:
                  - providerSpec
                  type: object
              type: object
          required:
          - selector
          type: object
        status:
          properties:
            availableReplicas:
              format: int32
              type: integer
            errorMessage:
              type: string
            errorReason:
              type: string
            fullyLabeledReplicas:
              format: int32
              type: integer
            observedGeneration:
              format: int64
              type: integer
            readyReplicas:
              format: int32
              type: integer
            replicas:
              format: int32
              type: integer
          required:
          - replicas
          type: object
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
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
        image: gcr.io/k8s-cluster-api/cluster-api-controller:0.0.0-alpha.4
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

type setupConfig struct {
	OS                  string
	OSFamily            string
	KubeletVersion      string
	ControlPlaneVersion string
}

var machineSetupConfig = `
items:
- machineParams:
  - os: {{ .OS }}
    roles:
    - Master
    versions:
      kubelet: {{ .KubeletVersion }}
      controlPlane: {{ .ControlPlaneVersion }}
  image: projects/ubuntu-os-cloud/global/images/family/{{ .OSFamily }}
  metadata:
    startupScript: |
      set -e
      set -x
      (
      ARCH=amd64

      function curl_metadata() {
          curl  --retry 5 --silent --fail --header "Metadata-Flavor: Google" "http://metadata/computeMetadata/v1/instance/$@"
      }

      function copy_file () {
          if ! curl_metadata attributes/$1; then
              return
          fi
          echo "Copying metadata $1 -> $2..."
          mkdir -p $(dirname $2)
          curl_metadata attributes/$1 > $2
          chmod $3 $2
      }

      curl -sf https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
      touch /etc/apt/sources.list.d/kubernetes.list
      sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'
      apt-get update -y
      apt-get install -y \
        socat \
        ebtables \
        apt-transport-https \
        cloud-utils \
        prips

      function install_configure_docker () {
        # prevent docker from auto-starting
        echo "exit 101" > /usr/sbin/policy-rc.d
        chmod +x /usr/sbin/policy-rc.d
        trap "rm /usr/sbin/policy-rc.d" RETURN
        apt-get install -y docker.io
        echo 'DOCKER_OPTS="--iptables=false --ip-masq=false"' > /etc/default/docker
        systemctl daemon-reload
        systemctl enable docker
        systemctl start docker
      }
      install_configure_docker

      curl -fsSL https://dl.k8s.io/release/${VERSION}/bin/linux/${ARCH}/kubeadm > /usr/bin/kubeadm.dl
      chmod a+rx /usr/bin/kubeadm.dl
      # kubeadm uses 10th IP as DNS server
      CLUSTER_DNS_SERVER=$(prips ${SERVICE_CIDR} | head -n 11 | tail -n 1)
      # Our Debian packages have versions like "1.8.0-00" or "1.8.0-01". Do a prefix
      # search based on our SemVer to find the right (newest) package version.
      function getversion() {
          name=$1
          prefix=$2
          prefix="${prefix//v}"
          version=$(apt-cache madison $name | awk '{ print $3 }' | grep ^$prefix | head -n1)
          if [[ -z "$version" ]]; then
              echo Can\'t find package $name with prefix $prefix
              exit 1
          fi
          echo $version
      }
      KUBELET=$(getversion kubelet ${KUBELET_VERSION}-)
      KUBEADM=$(getversion kubeadm ${KUBELET_VERSION}-)
      apt-get install -y \
          kubelet=${KUBELET} \
          kubeadm=${KUBEADM}
      mv /usr/bin/kubeadm.dl /usr/bin/kubeadm
      chmod a+rx /usr/bin/kubeadm

      # Override network args to use kubenet instead of cni, override Kubelet DNS args and
      # add cloud provider args.
      cat > /etc/default/kubelet <<EOF
      KUBELET_EXTRA_ARGS="--network-plugin=kubenet --cluster-dns=${CLUSTER_DNS_SERVER} --cluster-domain=${CLUSTER_DNS_DOMAIN} --cloud-provider=gce --cloud-config=/etc/kubernetes/cloud-config"
      EOF

      systemctl daemon-reload
      systemctl restart kubelet.service
      PRIVATEIP='curl_metadata "network-interfaces/0/ip"'
      echo $PRIVATEIP > /tmp/.ip
      PUBLICIP='curl_metadata "network-interfaces/0/access-configs/0/external-ip"'

      # Set up the GCE cloud config, which gets picked up by kubeadm init since cloudProvider is set to GCE.
      copy_file cloud-config /etc/kubernetes/cloud-config 0644

      # Set up kubeadm config file to pass parameters to kubeadm init.
      cat > /etc/kubernetes/kubeadm_config.yaml <<EOF
      apiVersion: kubeadm.k8s.io/v1alpha2
      kind: MasterConfiguration
      api:
        advertiseAddress: ${PUBLICIP}
        bindPort: ${PORT}
      networking:
        serviceSubnet: ${SERVICE_CIDR}
      kubernetesVersion: v${CONTROL_PLANE_VERSION}
      apiServerCertSANs:
      - ${PUBLICIP}
      - ${PRIVATEIP}
      bootstrapTokens:
      - groups:
        - system:bootstrappers:kubeadm:default-node-token
        token: ${TOKEN}
      apiServerExtraArgs:
        cloud-provider: gce
      controllerManagerExtraArgs:
        allocate-node-cidrs: "true"
        cloud-provider: gce
        cluster-cidr: ${POD_CIDR}
        service-cluster-ip-range: ${SERVICE_CIDR}
      EOF

      function install_certificates () {
          if ! curl_metadata "attributes/ca-cert"; then
              return
          fi
          echo "Configuring custom certificate authority..."
          PKI_PATH=/etc/kubernetes/pki
          mkdir -p ${PKI_PATH}
          CA_CERT_PATH=${PKI_PATH}/ca.crt
          curl_metadata "attributes/ca-cert" | base64 -d > ${CA_CERT_PATH}
          chmod 0644 ${CA_CERT_PATH}
          CA_KEY_PATH=${PKI_PATH}/ca.key
          curl_metadata "attributes/ca-key" | base64 -d > ${CA_KEY_PATH}
          chmod 0600 ${CA_KEY_PATH}
      }

      # Create and set bridge-nf-call-iptables to 1 to pass the kubeadm preflight check.
      # Workaround was found here:
      # http://zeeshanali.com/sysadmin/fixed-sysctl-cannot-stat-procsysnetbridgebridge-nf-call-iptables/
      modprobe br_netfilter

      install_certificates

      kubeadm init --config /etc/kubernetes/kubeadm_config.yaml

      for tries in $(seq 1 60); do
          kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname) machine=${MACHINE} && break
          sleep 1
      done
      echo done.
      ) 2>&1 | tee /var/log/startup.log
- machineParams:
  - os: {{ .OS }}
    roles:
    - Node
    versions:
      kubelet: {{ .KubeletVersion }}
  image: projects/ubuntu-os-cloud/global/images/family/{{ .OSFamily }}
  metadata:
    startupScript: |
      set -e
      set -x
      (
      function curl_metadata() {
          curl  --retry 5 --silent --fail --header "Metadata-Flavor: Google" "http://metadata/computeMetadata/v1/instance/$@"
      }

      function copy_file () {
          if ! curl_metadata attributes/$1; then
              return
          fi
          echo "Copying metadata $1 -> $2..."
          mkdir -p $(dirname $2)
          curl_metadata attributes/$1 > $2
          chmod $3 $2
      }

      apt-get update
      apt-get install -y apt-transport-https prips
      apt-key adv --keyserver hkp://keyserver.ubuntu.com --recv-keys F76221572C52609D
      cat <<EOF > /etc/apt/sources.list.d/k8s.list
      deb [arch=amd64] https://apt.dockerproject.org/repo ubuntu-xenial main
      EOF
      apt-get update

      function install_configure_docker () {
          # prevent docker from auto-starting
          echo "exit 101" > /usr/sbin/policy-rc.d
          chmod +x /usr/sbin/policy-rc.d
          trap "rm /usr/sbin/policy-rc.d" RETURN
          apt-get install -y docker-engine=1.12.0-0~xenial
          echo 'DOCKER_OPTS="--iptables=false --ip-masq=false"' > /etc/default/docker
          systemctl daemon-reload
          systemctl enable docker
          systemctl start docker
      }

      install_configure_docker

      copy_file cloud-config /etc/kubernetes/cloud-config 0644

      curl -fs https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
      cat <<EOF > /etc/apt/sources.list.d/kubernetes.list
      deb http://apt.kubernetes.io/ kubernetes-xenial main
      EOF
      apt-get update

      # Our Debian packages have versions like "1.8.0-00" or "1.8.0-01". Do a prefix
      # search based on our SemVer to find the right (newest) package version.
      function getversion() {
      	name=$1
      	prefix=$2
      	prefix="${prefix//v}"
      	version=$(apt-cache madison $name | awk '{ print $3 }' | grep ^$prefix | head -n1)
      	if [[ -z "$version" ]]; then
      		echo Can\'t find package $name with prefix $prefix
      		exit 1
      	fi
      	echo $version
      }
      KUBELET=$(getversion kubelet ${KUBELET_VERSION}-)
      KUBEADM=$(getversion kubeadm ${KUBELET_VERSION}-)
      KUBECTL=$(getversion kubectl ${KUBELET_VERSION}-)
      apt-get install -y kubelet=${KUBELET} kubeadm=${KUBEADM} kubectl=${KUBECTL}
      # kubeadm uses 10th IP as DNS server
      CLUSTER_DNS_SERVER=$(prips ${SERVICE_CIDR} | head -n 11 | tail -n 1)
      # Override network args to use kubenet instead of cni, override Kubelet DNS args and
      # add cloud provider args.
      cat > /etc/default/kubelet <<EOF
      KUBELET_EXTRA_ARGS="--network-plugin=kubenet --cluster-dns=${CLUSTER_DNS_SERVER} --cluster-domain=${CLUSTER_DNS_DOMAIN} --cloud-provider=gce --cloud-config=/etc/kubernetes/cloud-config"
      EOF

      systemctl daemon-reload
      systemctl restart kubelet.service
      kubeadm join --token "${TOKEN}" "${MASTER}" --ignore-preflight-errors=all --discovery-token-unsafe-skip-ca-verification
      for tries in $(seq 1 60); do
      	kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname) machine=${MACHINE} && break
      	sleep 1
      done
      echo done.
      ) 2>&1 | tee /var/log/startup.log
`

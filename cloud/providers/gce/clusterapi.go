package gce

import (
	"bytes"
	"encoding/base64"
	"strings"
	"text/template"

	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	yaml "gopkg.in/yaml.v2"
)

func (conn *cloudConnector) getControllerManager() (string, error) {
	config := conn.cluster.ClusterConfig()
	cred, err := Store(conn.ctx).Owner(conn.owner).Credentials().Get(config.CredentialName)
	if err != nil {
		return "", err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}

	machineSetupConfig, err := getMachineSetupConfig(config)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("controller-manager-config").Parse(ControllerManager)
	if err != nil {
		return "", err
	}

	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, controllerManagerConfig{
		MachineConfig:   machineSetupConfig,
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

func getMachineSetupConfig(config *api.ClusterConfig) (string, error) {
	tmpl, err := template.New("machine-config").Parse(machineSetupConfig)
	if err != nil {
		return "", err
	}
	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, setupConfig{
		OS:                        config.Cloud.InstanceImage,
		OSFamily:                  config.Cloud.OS,
		KubernetesVersion:         strings.TrimPrefix(config.KubernetesVersion, "v"),
		ControlPlaneStartupScript: "",
		NodeStartupScript:         "",
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
        adminKubeconfig:
          description: AdminKubeconfig generated using the certificates part of the
            spec do not move to status, since it uses on disk ca certs, which causes
            issues during regeneration
          type: string
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        caKeyPair:
          description: CAKeyPair is the key pair for CA certs.
          properties:
            cert:
              description: base64 encoded cert and key
              format: byte
              type: string
            key:
              format: byte
              type: string
          required:
          - cert
          - key
          type: object
        clusterConfiguration:
          description: ClusterConfiguration holds the cluster-wide information used
            during a kubeadm init call.
          properties:
            apiServer:
              description: APIServer contains extra settings for the API server control
                plane component
              properties:
                certSANs:
                  description: CertSANs sets extra Subject Alternative Names for the
                    API Server signing cert.
                  items:
                    type: string
                  type: array
                extraArgs:
                  description: 'ExtraArgs is an extra set of flags to pass to the
                    control plane component. TODO: This is temporary and ideally we
                    would like to switch all components to use ComponentConfig + ConfigMaps.'
                  type: object
                extraVolumes:
                  description: ExtraVolumes is an extra set of host volumes, mounted
                    to the control plane component.
                  items:
                    properties:
                      hostPath:
                        description: HostPath is the path in the host that will be
                          mounted inside the pod.
                        type: string
                      mountPath:
                        description: MountPath is the path inside the pod where hostPath
                          will be mounted.
                        type: string
                      name:
                        description: Name of the volume inside the pod template.
                        type: string
                      pathType:
                        description: PathType is the type of the HostPath.
                        type: string
                      readOnly:
                        description: ReadOnly controls write access to the volume
                        type: boolean
                    required:
                    - name
                    - hostPath
                    - mountPath
                    type: object
                  type: array
                timeoutForControlPlane:
                  description: TimeoutForControlPlane controls the timeout that we
                    use for API server to appear
                  type: object
              type: object
            apiVersion:
              description: 'APIVersion defines the versioned schema of this representation
                of an object. Servers should convert recognized schemas to the latest
                internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
              type: string
            certificatesDir:
              description: CertificatesDir specifies where to store or look for all
                required certificates.
              type: string
            clusterName:
              description: The cluster name
              type: string
            controlPlaneEndpoint:
              description: 'ControlPlaneEndpoint sets a stable IP address or DNS name
                for the control plane; it can be a valid IP address or a RFC-1123
                DNS subdomain, both with optional TCP port. In case the ControlPlaneEndpoint
                is not specified, the AdvertiseAddress + BindPort are used; in case
                the ControlPlaneEndpoint is specified but without a TCP port, the
                BindPort is used. Possible usages are: e.g. In a cluster with more
                than one control plane instances, this field should be assigned the
                address of the external load balancer in front of the control plane
                instances. e.g.  in environments with enforced node recycling, the
                ControlPlaneEndpoint could be used for assigning a stable DNS to the
                control plane.'
              type: string
            controllerManager:
              description: ControllerManager contains extra settings for the controller
                manager control plane component
              properties:
                extraArgs:
                  description: 'ExtraArgs is an extra set of flags to pass to the
                    control plane component. TODO: This is temporary and ideally we
                    would like to switch all components to use ComponentConfig + ConfigMaps.'
                  type: object
                extraVolumes:
                  description: ExtraVolumes is an extra set of host volumes, mounted
                    to the control plane component.
                  items:
                    properties:
                      hostPath:
                        description: HostPath is the path in the host that will be
                          mounted inside the pod.
                        type: string
                      mountPath:
                        description: MountPath is the path inside the pod where hostPath
                          will be mounted.
                        type: string
                      name:
                        description: Name of the volume inside the pod template.
                        type: string
                      pathType:
                        description: PathType is the type of the HostPath.
                        type: string
                      readOnly:
                        description: ReadOnly controls write access to the volume
                        type: boolean
                    required:
                    - name
                    - hostPath
                    - mountPath
                    type: object
                  type: array
              type: object
            dns:
              description: DNS defines the options for the DNS add-on installed in
                the cluster.
              properties:
                imageRepository:
                  description: ImageRepository sets the container registry to pull
                    images from. if not set, the ImageRepository defined in ClusterConfiguration
                    will be used instead.
                  type: string
                imageTag:
                  description: ImageTag allows to specify a tag for the image. In
                    case this value is set, kubeadm does not change automatically
                    the version of the above components during upgrades.
                  type: string
                type:
                  description: Type defines the DNS add-on to be used
                  type: string
              required:
              - type
              type: object
            etcd:
              description: Etcd holds configuration for etcd.
              properties:
                external:
                  description: External describes how to connect to an external etcd
                    cluster Local and External are mutually exclusive
                  properties:
                    caFile:
                      description: CAFile is an SSL Certificate Authority file used
                        to secure etcd communication. Required if using a TLS connection.
                      type: string
                    certFile:
                      description: CertFile is an SSL certification file used to secure
                        etcd communication. Required if using a TLS connection.
                      type: string
                    endpoints:
                      description: Endpoints of etcd members. Required for ExternalEtcd.
                      items:
                        type: string
                      type: array
                    keyFile:
                      description: KeyFile is an SSL key file used to secure etcd
                        communication. Required if using a TLS connection.
                      type: string
                  required:
                  - endpoints
                  - caFile
                  - certFile
                  - keyFile
                  type: object
                local:
                  description: Local provides configuration knobs for configuring
                    the local etcd instance Local and External are mutually exclusive
                  properties:
                    dataDir:
                      description: DataDir is the directory etcd will place its data.
                        Defaults to "/var/lib/etcd".
                      type: string
                    extraArgs:
                      description: ExtraArgs are extra arguments provided to the etcd
                        binary when run inside a static pod.
                      type: object
                    imageRepository:
                      description: ImageRepository sets the container registry to
                        pull images from. if not set, the ImageRepository defined
                        in ClusterConfiguration will be used instead.
                      type: string
                    imageTag:
                      description: ImageTag allows to specify a tag for the image.
                        In case this value is set, kubeadm does not change automatically
                        the version of the above components during upgrades.
                      type: string
                    peerCertSANs:
                      description: PeerCertSANs sets extra Subject Alternative Names
                        for the etcd peer signing cert.
                      items:
                        type: string
                      type: array
                    serverCertSANs:
                      description: ServerCertSANs sets extra Subject Alternative Names
                        for the etcd server signing cert.
                      items:
                        type: string
                      type: array
                  required:
                  - dataDir
                  type: object
              type: object
            featureGates:
              description: FeatureGates enabled by the user.
              type: object
            imageRepository:
              description: ImageRepository sets the container registry to pull images
                from. If empty, k8s.gcr.io will be used by default; in case of kubernetes
                version is a CI build (kubernetes version starts with ci/ or ci-cross/)
                gcr.io/kubernetes-ci-images will be used as a default for control
                plane components and for kube-proxy, while k8s.gcr.io will be used
                for all the other images.
              type: string
            kind:
              description: 'Kind is a string value representing the REST resource
                this object represents. Servers may infer this from the endpoint the
                client submits requests to. Cannot be updated. In CamelCase. More
                info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
              type: string
            kubernetesVersion:
              description: KubernetesVersion is the target version of the control
                plane.
              type: string
            networking:
              description: Networking holds configuration for the networking topology
                of the cluster.
              properties:
                dnsDomain:
                  description: DNSDomain is the dns domain used by k8s services. Defaults
                    to "cluster.local".
                  type: string
                podSubnet:
                  description: PodSubnet is the subnet used by pods.
                  type: string
                serviceSubnet:
                  description: ServiceSubnet is the subnet used by k8s services. Defaults
                    to "10.96.0.0/12".
                  type: string
              required:
              - serviceSubnet
              - podSubnet
              - dnsDomain
              type: object
            scheduler:
              description: Scheduler contains extra settings for the scheduler control
                plane component
              properties:
                extraArgs:
                  description: 'ExtraArgs is an extra set of flags to pass to the
                    control plane component. TODO: This is temporary and ideally we
                    would like to switch all components to use ComponentConfig + ConfigMaps.'
                  type: object
                extraVolumes:
                  description: ExtraVolumes is an extra set of host volumes, mounted
                    to the control plane component.
                  items:
                    properties:
                      hostPath:
                        description: HostPath is the path in the host that will be
                          mounted inside the pod.
                        type: string
                      mountPath:
                        description: MountPath is the path inside the pod where hostPath
                          will be mounted.
                        type: string
                      name:
                        description: Name of the volume inside the pod template.
                        type: string
                      pathType:
                        description: PathType is the type of the HostPath.
                        type: string
                      readOnly:
                        description: ReadOnly controls write access to the volume
                        type: boolean
                    required:
                    - name
                    - hostPath
                    - mountPath
                    type: object
                  type: array
              type: object
            useHyperKubeImage:
              description: UseHyperKubeImage controls if hyperkube should be used
                for Kubernetes components instead of their respective separate images
              type: boolean
          required:
          - etcd
          - networking
          - kubernetesVersion
          - controlPlaneEndpoint
          - dns
          - certificatesDir
          - imageRepository
          type: object
        discoveryHashes:
          description: DiscoveryHashes generated using the certificates part of the
            spec, used by master and nodes bootstrapping this never changes until
            ca is rotated do not move to status, since it uses on disk ca certs, which
            causes issues during regeneration
          items:
            type: string
          type: array
        etcdCAKeyPair:
          description: EtcdCAKeyPair is the key pair for etcd.
          properties:
            cert:
              description: base64 encoded cert and key
              format: byte
              type: string
            key:
              format: byte
              type: string
          required:
          - cert
          - key
          type: object
        frontProxyCAKeyPair:
          description: FrontProxyCAKeyPair is the key pair for the front proxy.
          properties:
            cert:
              description: base64 encoded cert and key
              format: byte
              type: string
            key:
              format: byte
              type: string
          required:
          - cert
          - key
          type: object
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        project:
          type: string
        saKeyPair:
          description: SAKeyPair is the service account key pair.
          properties:
            cert:
              description: base64 encoded cert and key
              format: byte
              type: string
            key:
              format: byte
              type: string
          required:
          - cert
          - key
          type: object
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
  name: gceclusterproviderstatuses.gceproviderconfig.k8s.io
spec:
  group: gceproviderconfig.k8s.io
  names:
    kind: GCEClusterProviderStatus
    plural: gceclusterproviderstatuses
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
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
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
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
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        machineType:
          type: string
        metadata:
          type: object
        os:
          description: The name of the OS to be installed on the machine.
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
  - gceclusterproviderstatuses
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
  - clusters/status
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
        image: pharmer/gce-controller:latest
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
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          properties:
            clusterNetwork:
              description: Cluster network configuration
              properties:
                pods:
                  description: The network ranges from which Pod networks are allocated.
                  properties:
                    cidrBlocks:
                      items:
                        type: string
                      type: array
                  required:
                  - cidrBlocks
                  type: object
                serviceDomain:
                  description: Domain name for services.
                  type: string
                services:
                  description: The network ranges from which service VIPs are allocated.
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
              description: Provider-specific serialized configuration to use during
                cluster creation. It is recommended that providers maintain their
                own versioned API types that should be serialized/deserialized from
                this field.
              properties:
                value:
                  description: Value is an inlined, serialized representation of the
                    resource configuration. It is recommended that providers maintain
                    their own versioned API types that should be serialized/deserialized
                    from this field, akin to component config.
                  type: object
                valueFrom:
                  description: Source for the provider configuration. Cannot be used
                    if value is not empty.
                  properties:
                    machineClass:
                      description: The machine class from which the provider config
                        should be sourced.
                      properties:
                        provider:
                          description: Provider is the name of the cloud-provider
                            which MachineClass is intended for.
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
              description: APIEndpoint represents the endpoint to communicate with
                the IP.
              items:
                properties:
                  host:
                    description: The hostname on which the API server is serving.
                    type: string
                  port:
                    description: The port on which the API server is serving.
                    format: int64
                    type: integer
                required:
                - host
                - port
                type: object
              type: array
            errorMessage:
              description: If set, indicates that there is a problem reconciling the
                state, and will be set to a descriptive error message.
              type: string
            errorReason:
              description: If set, indicates that there is a problem reconciling the
                state, and will be set to a token value suitable for programmatic
                interpretation.
              type: string
            providerStatus:
              description: Provider-specific status. It is recommended that providers
                maintain their own versioned API types that should be serialized/deserialized
                from this field.
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
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        providerSpec:
          description: Provider-specific configuration to use during node creation.
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
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          properties:
            minReadySeconds:
              description: Minimum number of seconds for which a newly created machine
                should be ready. Defaults to 0 (machine will be considered available
                as soon as it is ready)
              format: int32
              type: integer
            paused:
              description: Indicates that the deployment is paused.
              type: boolean
            progressDeadlineSeconds:
              description: The maximum time in seconds for a deployment to make progress
                before it is considered to be failed. The deployment controller will
                continue to process failed deployments and a condition with a ProgressDeadlineExceeded
                reason will be surfaced in the deployment status. Note that progress
                will not be estimated during the time a deployment is paused. Defaults
                to 600s.
              format: int32
              type: integer
            replicas:
              description: Number of desired machines. Defaults to 1. This is a pointer
                to distinguish between explicit zero and not specified.
              format: int32
              type: integer
            revisionHistoryLimit:
              description: The number of old MachineSets to retain to allow rollback.
                This is a pointer to distinguish between explicit zero and not specified.
                Defaults to 1.
              format: int32
              type: integer
            selector:
              description: Label selector for machines. Existing MachineSets whose
                machines are selected by this will be the ones affected by this deployment.
                It must match the machine template's labels.
              type: object
            strategy:
              description: The deployment strategy to use to replace existing machines
                with new ones.
              properties:
                rollingUpdate:
                  description: Rolling update config params. Present only if MachineDeploymentStrategyType
                    = RollingUpdate.
                  properties:
                    maxSurge:
                      description: 'The maximum number of machines that can be scheduled
                        above the desired number of machines. Value can be an absolute
                        number (ex: 5) or a percentage of desired machines (ex: 10%).
                        This can not be 0 if MaxUnavailable is 0. Absolute number
                        is calculated from percentage by rounding up. Defaults to
                        1. Example: when this is set to 30%, the new MachineSet can
                        be scaled up immediately when the rolling update starts, such
                        that the total number of old and new machines do not exceed
                        130% of desired machines. Once old machines have been killed,
                        new MachineSet can be scaled up further, ensuring that total
                        number of machines running at any time during the update is
                        at most 130% of desired machines.'
                      oneOf:
                      - type: string
                      - type: integer
                    maxUnavailable:
                      description: 'The maximum number of machines that can be unavailable
                        during the update. Value can be an absolute number (ex: 5)
                        or a percentage of desired machines (ex: 10%). Absolute number
                        is calculated from percentage by rounding down. This can not
                        be 0 if MaxSurge is 0. Defaults to 0. Example: when this is
                        set to 30%, the old MachineSet can be scaled down to 70% of
                        desired machines immediately when the rolling update starts.
                        Once new machines are ready, old MachineSet can be scaled
                        down further, followed by scaling up the new MachineSet, ensuring
                        that the total number of machines available at all times during
                        the update is at least 70% of desired machines.'
                      oneOf:
                      - type: string
                      - type: integer
                  type: object
                type:
                  description: Type of deployment. Currently the only supported strategy
                    is "RollingUpdate". Default is RollingUpdate.
                  type: string
              type: object
            template:
              description: Template describes the machines that will be created.
              properties:
                metadata:
                  description: 'Standard object''s metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata'
                  type: object
                spec:
                  description: 'Specification of the desired behavior of the machine.
                    More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status'
                  properties:
                    configSource:
                      description: ConfigSource is used to populate in the associated
                        Node for dynamic kubelet config. This field already exists
                        in Node, so any updates to it in the Machine spec will be
                        automatically copied to the linked NodeRef from the status.
                        The rest of dynamic kubelet config support should then work
                        as-is.
                      type: object
                    metadata:
                      description: ObjectMeta will autopopulate the Node created.
                        Use this to indicate what labels, annotations, name prefix,
                        etc., should be used when creating the Node.
                      type: object
                    providerID:
                      description: ProviderID is the identification ID of the machine
                        provided by the provider. This field must match the provider
                        ID as seen on the node object corresponding to this machine.
                        This field is required by higher level consumers of cluster-api.
                        Example use case is cluster autoscaler with cluster-api as
                        provider. Clean-up login in the autoscaler compares machines
                        v/s nodes to find out machines at provider which could not
                        get registered as Kubernetes nodes. With cluster-api as a
                        generic out-of-tree provider for autoscaler, this field is
                        required by autoscaler to be able to have a provider view
                        of the list of machines. Another list of nodes is queries
                        from the k8s apiserver and then comparison is done to find
                        out unregistered machines and are marked for delete. This
                        field will be set by the actuators and consumed by higher
                        level entities like autoscaler  who will be interfacing with
                        cluster-api as generic provider.
                      type: string
                    providerSpec:
                      description: ProviderSpec details Provider-specific configuration
                        to use during node creation.
                      properties:
                        value:
                          description: Value is an inlined, serialized representation
                            of the resource configuration. It is recommended that
                            providers maintain their own versioned API types that
                            should be serialized/deserialized from this field, akin
                            to component config.
                          type: object
                        valueFrom:
                          description: Source for the provider configuration. Cannot
                            be used if value is not empty.
                          properties:
                            machineClass:
                              description: The machine class from which the provider
                                config should be sourced.
                              properties:
                                provider:
                                  description: Provider is the name of the cloud-provider
                                    which MachineClass is intended for.
                                  type: string
                              type: object
                          type: object
                      type: object
                    taints:
                      description: Taints is the full, authoritative list of taints
                        to apply to the corresponding Node. This list will overwrite
                        any modifications made to the Node on an ongoing basis.
                      items:
                        type: object
                      type: array
                    versions:
                      description: Versions of key software to use. This field is
                        optional at cluster creation time, and omitting the field
                        indicates that the cluster installation tool should select
                        defaults for the user. These defaults may differ based on
                        the cluster installer, but the tool should populate the values
                        it uses when persisting Machine objects. A Machine spec missing
                        this field at runtime is invalid.
                      properties:
                        controlPlane:
                          description: ControlPlane is the semantic version of the
                            Kubernetes control plane to run. This should only be populated
                            when the machine is a control plane.
                          type: string
                        kubelet:
                          description: Kubelet is the semantic version of kubelet
                            to run
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
              description: Total number of available machines (ready for at least
                minReadySeconds) targeted by this deployment.
              format: int32
              type: integer
            observedGeneration:
              description: The generation observed by the deployment controller.
              format: int64
              type: integer
            readyReplicas:
              description: Total number of ready machines targeted by this deployment.
              format: int32
              type: integer
            replicas:
              description: Total number of non-terminated machines targeted by this
                deployment (their labels match the selector).
              format: int32
              type: integer
            unavailableReplicas:
              description: Total number of unavailable machines targeted by this deployment.
                This is the total number of machines that are still required for the
                deployment to have 100% available capacity. They may either be machines
                that are running but not yet available or machines that still have
                not been created.
              format: int32
              type: integer
            updatedReplicas:
              description: Total number of non-terminated machines targeted by this
                deployment that have the desired template spec.
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
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          properties:
            configSource:
              description: ConfigSource is used to populate in the associated Node
                for dynamic kubelet config. This field already exists in Node, so
                any updates to it in the Machine spec will be automatically copied
                to the linked NodeRef from the status. The rest of dynamic kubelet
                config support should then work as-is.
              type: object
            metadata:
              description: ObjectMeta will autopopulate the Node created. Use this
                to indicate what labels, annotations, name prefix, etc., should be
                used when creating the Node.
              type: object
            providerID:
              description: ProviderID is the identification ID of the machine provided
                by the provider. This field must match the provider ID as seen on
                the node object corresponding to this machine. This field is required
                by higher level consumers of cluster-api. Example use case is cluster
                autoscaler with cluster-api as provider. Clean-up login in the autoscaler
                compares machines v/s nodes to find out machines at provider which
                could not get registered as Kubernetes nodes. With cluster-api as
                a generic out-of-tree provider for autoscaler, this field is required
                by autoscaler to be able to have a provider view of the list of machines.
                Another list of nodes is queries from the k8s apiserver and then comparison
                is done to find out unregistered machines and are marked for delete.
                This field will be set by the actuators and consumed by higher level
                entities like autoscaler  who will be interfacing with cluster-api
                as generic provider.
              type: string
            providerSpec:
              description: ProviderSpec details Provider-specific configuration to
                use during node creation.
              properties:
                value:
                  description: Value is an inlined, serialized representation of the
                    resource configuration. It is recommended that providers maintain
                    their own versioned API types that should be serialized/deserialized
                    from this field, akin to component config.
                  type: object
                valueFrom:
                  description: Source for the provider configuration. Cannot be used
                    if value is not empty.
                  properties:
                    machineClass:
                      description: The machine class from which the provider config
                        should be sourced.
                      properties:
                        provider:
                          description: Provider is the name of the cloud-provider
                            which MachineClass is intended for.
                          type: string
                      type: object
                  type: object
              type: object
            taints:
              description: Taints is the full, authoritative list of taints to apply
                to the corresponding Node. This list will overwrite any modifications
                made to the Node on an ongoing basis.
              items:
                type: object
              type: array
            versions:
              description: Versions of key software to use. This field is optional
                at cluster creation time, and omitting the field indicates that the
                cluster installation tool should select defaults for the user. These
                defaults may differ based on the cluster installer, but the tool should
                populate the values it uses when persisting Machine objects. A Machine
                spec missing this field at runtime is invalid.
              properties:
                controlPlane:
                  description: ControlPlane is the semantic version of the Kubernetes
                    control plane to run. This should only be populated when the machine
                    is a control plane.
                  type: string
                kubelet:
                  description: Kubelet is the semantic version of kubelet to run
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
              description: Addresses is a list of addresses assigned to the machine.
                Queried from cloud provider, if available.
              items:
                type: object
              type: array
            conditions:
              description: 'Conditions lists the conditions synced from the node conditions
                of the corresponding node-object. Machine-controller is responsible
                for keeping conditions up-to-date. MachineSet controller will be taking
                these conditions as a signal to decide if machine is healthy or needs
                to be replaced. Refer: https://kubernetes.io/docs/concepts/architecture/nodes/#condition'
              items:
                type: object
              type: array
            errorMessage:
              description: ErrorMessage will be set in the event that there is a terminal
                problem reconciling the Machine and will contain a more verbose string
                suitable for logging and human consumption.  This field should not
                be set for transitive errors that a controller faces that are expected
                to be fixed automatically over time (like service outages), but instead
                indicate that something is fundamentally wrong with the Machine's
                spec or the configuration of the controller, and that manual intervention
                is required. Examples of terminal errors would be invalid combinations
                of settings in the spec, values that are unsupported by the controller,
                or the responsible controller itself being critically misconfigured.  Any
                transient errors that occur during the reconciliation of Machines
                can be added as events to the Machine object and/or logged in the
                controller's output.
              type: string
            errorReason:
              description: ErrorReason will be set in the event that there is a terminal
                problem reconciling the Machine and will contain a succinct value
                suitable for machine interpretation.  This field should not be set
                for transitive errors that a controller faces that are expected to
                be fixed automatically over time (like service outages), but instead
                indicate that something is fundamentally wrong with the Machine's
                spec or the configuration of the controller, and that manual intervention
                is required. Examples of terminal errors would be invalid combinations
                of settings in the spec, values that are unsupported by the controller,
                or the responsible controller itself being critically misconfigured.  Any
                transient errors that occur during the reconciliation of Machines
                can be added as events to the Machine object and/or logged in the
                controller's output.
              type: string
            lastOperation:
              description: LastOperation describes the last-operation performed by
                the machine-controller. This API should be useful as a history in
                terms of the latest operation performed on the specific machine. It
                should also convey the state of the latest-operation for example if
                it is still on-going, failed or completed successfully.
              properties:
                description:
                  description: Description is the human-readable description of the
                    last operation.
                  type: string
                lastUpdated:
                  description: LastUpdated is the timestamp at which LastOperation
                    API was last-updated.
                  format: date-time
                  type: string
                state:
                  description: State is the current status of the last performed operation.
                    E.g. Processing, Failed, Successful etc
                  type: string
                type:
                  description: Type is the type of operation which was last performed.
                    E.g. Create, Delete, Update etc
                  type: string
              type: object
            lastUpdated:
              description: LastUpdated identifies when this status was last observed.
              format: date-time
              type: string
            nodeRef:
              description: NodeRef will point to the corresponding Node if it exists.
              type: object
            phase:
              description: Phase represents the current phase of machine actuation.
                E.g. Pending, Running, Terminating, Failed etc.
              type: string
            providerStatus:
              description: ProviderStatus details a Provider-specific status. It is
                recommended that providers maintain their own versioned API types
                that should be serialized/deserialized from this field.
              type: object
            versions:
              description: 'Versions specifies the current versions of software on
                the corresponding Node (if it exists). This is provided for a few
                reasons:  1) It is more convenient than checking the NodeRef, traversing
                it to    the Node, and finding the appropriate field in Node.Status.NodeInfo    (which
                uses different field names and formatting). 2) It removes some of
                the dependency on the structure of the Node,    so that if the structure
                of Node.Status.NodeInfo changes, only    machine controllers need
                to be updated, rather than every client    of the Machines API. 3)
                There is no other simple way to check the control plane    version.
                A client would have to connect directly to the apiserver    running
                on the target node in order to find out its version.'
              properties:
                controlPlane:
                  description: ControlPlane is the semantic version of the Kubernetes
                    control plane to run. This should only be populated when the machine
                    is a control plane.
                  type: string
                kubelet:
                  description: Kubelet is the semantic version of kubelet to run
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
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          properties:
            minReadySeconds:
              description: MinReadySeconds is the minimum number of seconds for which
                a newly created machine should be ready. Defaults to 0 (machine will
                be considered available as soon as it is ready)
              format: int32
              type: integer
            replicas:
              description: Replicas is the number of desired replicas. This is a pointer
                to distinguish between explicit zero and unspecified. Defaults to
                1.
              format: int32
              type: integer
            selector:
              description: 'Selector is a label query over machines that should match
                the replica count. Label keys and values that must match in order
                to be controlled by this MachineSet. It must match the machine template''s
                labels. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors'
              type: object
            template:
              description: Template is the object that describes the machine that
                will be created if insufficient replicas are detected.
              properties:
                metadata:
                  description: 'Standard object''s metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata'
                  type: object
                spec:
                  description: 'Specification of the desired behavior of the machine.
                    More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status'
                  properties:
                    configSource:
                      description: ConfigSource is used to populate in the associated
                        Node for dynamic kubelet config. This field already exists
                        in Node, so any updates to it in the Machine spec will be
                        automatically copied to the linked NodeRef from the status.
                        The rest of dynamic kubelet config support should then work
                        as-is.
                      type: object
                    metadata:
                      description: ObjectMeta will autopopulate the Node created.
                        Use this to indicate what labels, annotations, name prefix,
                        etc., should be used when creating the Node.
                      type: object
                    providerID:
                      description: ProviderID is the identification ID of the machine
                        provided by the provider. This field must match the provider
                        ID as seen on the node object corresponding to this machine.
                        This field is required by higher level consumers of cluster-api.
                        Example use case is cluster autoscaler with cluster-api as
                        provider. Clean-up login in the autoscaler compares machines
                        v/s nodes to find out machines at provider which could not
                        get registered as Kubernetes nodes. With cluster-api as a
                        generic out-of-tree provider for autoscaler, this field is
                        required by autoscaler to be able to have a provider view
                        of the list of machines. Another list of nodes is queries
                        from the k8s apiserver and then comparison is done to find
                        out unregistered machines and are marked for delete. This
                        field will be set by the actuators and consumed by higher
                        level entities like autoscaler  who will be interfacing with
                        cluster-api as generic provider.
                      type: string
                    providerSpec:
                      description: ProviderSpec details Provider-specific configuration
                        to use during node creation.
                      properties:
                        value:
                          description: Value is an inlined, serialized representation
                            of the resource configuration. It is recommended that
                            providers maintain their own versioned API types that
                            should be serialized/deserialized from this field, akin
                            to component config.
                          type: object
                        valueFrom:
                          description: Source for the provider configuration. Cannot
                            be used if value is not empty.
                          properties:
                            machineClass:
                              description: The machine class from which the provider
                                config should be sourced.
                              properties:
                                provider:
                                  description: Provider is the name of the cloud-provider
                                    which MachineClass is intended for.
                                  type: string
                              type: object
                          type: object
                      type: object
                    taints:
                      description: Taints is the full, authoritative list of taints
                        to apply to the corresponding Node. This list will overwrite
                        any modifications made to the Node on an ongoing basis.
                      items:
                        type: object
                      type: array
                    versions:
                      description: Versions of key software to use. This field is
                        optional at cluster creation time, and omitting the field
                        indicates that the cluster installation tool should select
                        defaults for the user. These defaults may differ based on
                        the cluster installer, but the tool should populate the values
                        it uses when persisting Machine objects. A Machine spec missing
                        this field at runtime is invalid.
                      properties:
                        controlPlane:
                          description: ControlPlane is the semantic version of the
                            Kubernetes control plane to run. This should only be populated
                            when the machine is a control plane.
                          type: string
                        kubelet:
                          description: Kubelet is the semantic version of kubelet
                            to run
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
              description: The number of available replicas (ready for at least minReadySeconds)
                for this MachineSet.
              format: int32
              type: integer
            errorMessage:
              type: string
            errorReason:
              description: In the event that there is a terminal problem reconciling
                the replicas, both ErrorReason and ErrorMessage will be set. ErrorReason
                will be populated with a succinct value suitable for machine interpretation,
                while ErrorMessage will contain a more verbose string suitable for
                logging and human consumption.  These fields should not be set for
                transitive errors that a controller faces that are expected to be
                fixed automatically over time (like service outages), but instead
                indicate that something is fundamentally wrong with the MachineTemplate's
                spec or the configuration of the machine controller, and that manual
                intervention is required. Examples of terminal errors would be invalid
                combinations of settings in the spec, values that are unsupported
                by the machine controller, or the responsible machine controller itself
                being critically misconfigured.  Any transient errors that occur during
                the reconciliation of Machines can be added as events to the MachineSet
                object and/or logged in the controller's output.
              type: string
            fullyLabeledReplicas:
              description: The number of replicas that have labels matching the labels
                of the machine template of the MachineSet.
              format: int32
              type: integer
            observedGeneration:
              description: ObservedGeneration reflects the generation of the most
                recently observed MachineSet.
              format: int64
              type: integer
            readyReplicas:
              description: The number of ready replicas for this MachineSet. A machine
                is considered ready when the node has been created and is "Ready".
              format: int32
              type: integer
            replicas:
              description: Replicas is the most recently observed number of replicas.
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

type setupConfig struct {
	OS                        string
	OSFamily                  string
	KubernetesVersion         string
	ControlPlaneStartupScript string
	NodeStartupScript         string
}

var machineSetupConfig = `
items:
- machineParams:
  - os: {{ .OS }}
    roles:
    - Master
    versions:
      kubelet: {{ .KubernetesVersion }}
      controlPlane: {{ .KubernetesVersion }}
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
        apt-get update && apt-get install apt-transport-https ca-certificates curl software-properties-common
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

        add-apt-repository \
        "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
              $(lsb_release -cs) \
              stable"

        apt-get update && apt-get install -y docker-ce=18.06.2~ce~3-0~ubuntu
        cat > /etc/docker/daemon.json <<EOF
      {
        "exec-opts": ["native.cgroupdriver=systemd"],
        "log-driver": "json-file",
        "log-opts": {
          "max-size": "100m"
        },
        "storage-driver": "overlay2"
      }
      EOF

        mkdir -p /etc/systemd/system/docker.service.d

        systemctl daemon-reload
        systemctl restart docker
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

      PRIVATEIP=$(curl_metadata "network-interfaces/0/ip")
      echo $PRIVATEIP > /tmp/.ip

      # Set up the GCE cloud config, which gets picked up by kubeadm init since cloudProvider is set to GCE.
      copy_file cloud-config /etc/kubernetes/ccm/cloud-config 0644

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
        echo "Configuring custom certificate authority..."
        PKI_PATH=/etc/kubernetes/pki
        mkdir -p ${PKI_PATH}
        CA_CERT_PATH=${PKI_PATH}/ca.crt
        curl_metadata "attributes/ca-cert" | base64 -d > ${CA_CERT_PATH}
        chmod 0644 ${CA_CERT_PATH}
        CA_KEY_PATH=${PKI_PATH}/ca.key
        curl_metadata "attributes/ca-key" | base64 -d > ${CA_KEY_PATH}
        chmod 0600 ${CA_KEY_PATH}

        echo "Configuring frontproxy certificate authority..."
        CA_CERT_PATH=/etc/kubernetes/pki/front-proxy-ca.crt
        curl_metadata "attributes/fpca-cert" | base64 -d > ${CA_CERT_PATH}
        chmod 0644 ${CA_CERT_PATH}
        CA_KEY_PATH=/etc/kubernetes/pki/front-proxy-ca.key
        curl_metadata "attributes/fpca-key" | base64 -d > ${CA_KEY_PATH}
        chmod 0600 ${CA_KEY_PATH}

        echo "Configuring etcd certificate authority..."
        mkdir -p /etc/kubernetes/pki/etcd
        CA_CERT_PATH=/etc/kubernetes/pki/etcd/ca.crt
        curl_metadata "attributes/etcdca-cert" | base64 -d > ${CA_CERT_PATH}
        chmod 0644 ${CA_CERT_PATH}
        CA_KEY_PATH=/etc/kubernetes/pki/etcd/ca.key
        curl_metadata "attributes/etcdca-key" | base64 -d > ${CA_KEY_PATH}
        chmod 0600 ${CA_KEY_PATH}

        echo "Configuring service account certificate authority..."
        CA_CERT_PATH=/etc/kubernetes/pki/sa.pub
        curl_metadata "attributes/sa-cert" | base64 -d > ${CA_CERT_PATH}
        chmod 0644 ${CA_CERT_PATH}
        CA_KEY_PATH=/etc/kubernetes/pki/sa.key
        curl_metadata "attributes/sa-key" | base64 -d > ${CA_KEY_PATH}
        chmod 0600 ${CA_KEY_PATH}
      }

      # Create and set bridge-nf-call-iptables to 1 to pass the kubeadm preflight check.
      # Workaround was found here:
      # http://zeeshanali.com/sysadmin/fixed-sysctl-cannot-stat-procsysnetbridgebridge-nf-call-iptables/
      modprobe br_netfilter

      install_certificates

      if [ "$TOKEN" == "" ]; then
      cat >/tmp/kubeadm.yaml <<EOF
      ---
      apiVersion: kubeadm.k8s.io/v1beta1
      kind: ClusterConfiguration
      apiServer:
        certSANs:
          - "${LOADBALANCER_IP}"
          - "${PRIVATEIP}"
          - "${PUBLICIP}"
        extraArgs:
          cloud-provider: gce
      controllerManager:
        extraArgs:
          cloud-provider: gce
      controlPlaneEndpoint: "${LOADBALANCER_IP}:6443"
      clusterName: "${CLUSTER_NAME}"
      networking:
        dnsDomain: "${CLUSTER_DNS_DOMAIN}"
        podSubnet: "${POD_CIDR}"
        serviceSubnet: "${SERVICE_CIDR}"
      kubernetesVersion: "${CONTROL_PLANE_VERSION}"
      ---
      apiVersion: kubeadm.k8s.io/v1beta1
      kind: InitConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          cloud-provider: gce
      EOF

        kubeadm init --config /tmp/kubeadm.yaml

      else
        cat > /tmp/kubeadm-controlplane-join-config.yaml <<EOF
      ---
      apiVersion: kubeadm.k8s.io/v1beta1
      kind: JoinConfiguration
      discovery:
        bootstrapToken:
          token: "${TOKEN}"
          apiServerEndpoint: "${LOADBALANCER_IP}:6443"
          caCertHashes:
            - "${CACERTHASH}"
      nodeRegistration:
        kubeletExtraArgs:
          cloud-provider: gce
          cloud-config: /etc/kubernetes/ccm/cloud-config
      controlPlane:
        localAPIEndpoint:
          advertiseAddress: "${PRIVATEIP}"
          bindPort: 6443
      EOF
        kubeadm join --config /tmp/kubeadm-controlplane-join-config.yaml
      fi

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
      kubelet: {{ .KubernetesVersion }}
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
          apt-get install -y docker.io
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

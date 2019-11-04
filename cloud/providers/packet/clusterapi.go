/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package packet

const ControllerManager = `
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
  name: packet-provider-controller-manager-service
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
  name: packet-provider-controller-manager
  namespace: {{ .ControllerNamespace }}
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
      controller-tools.k8s.io: "1.0"
  serviceName: packet-provider-controller-manager-service
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
        - --cluster-name={{ .ClusterName }}
        - --kubeconfig=/etc/kubernetes/admin.conf 
        env:
        image: {{ .ControllerImage }}
        imagePullPolicy: Always        
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
        - name: etcd-certs
          mountPath: /root/.pharmer/store.d/clusters/{{ .ClusterName }}/pki/etcd
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
      - name: etcd-certs
        secret:
          secretName: pharmer-etcd
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

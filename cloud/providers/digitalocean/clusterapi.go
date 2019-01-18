package digitalocean

const ClusterAPIDOProviderComponentsTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: do-provider-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
    controller-tools.k8s.io: "1.0"
  name: do-provider-controller-manager-service
  namespace: do-provider-system
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
  namespace: do-provider-system
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
      containers:
      - args:
        - controller
        - --kubeconfig=/etc/kubernets/admin.conf 
        env:
        image: pharmer/machine-controller:clusterapi
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
          mountPath: /root/.pharmer/store.d/clusters/capi11/ssh
        - name: certificates
          mountPath: /root/.pharmer/store.d/clusters/capi11/pki
        - name: etcd-cert
          mountPath: /root/.pharmer/store.d/clusters/capi11/pki/etcd
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
`

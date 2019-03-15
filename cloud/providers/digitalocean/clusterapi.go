package digitalocean

const ControllerManager = `
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
        - name: credential
          mountPath: /root/.aws/credential
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
`

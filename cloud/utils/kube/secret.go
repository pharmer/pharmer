package kube

import (
	"github.com/appscode/go/log"
	"github.com/appscode/go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	api "pharmer.dev/pharmer/apis/v1alpha1"
)

func CreateCredentialSecret(client kubernetes.Interface, cloudProvider, namespace string, data map[string]string) error {
	newData := make(map[string][]byte)
	for key, value := range data {
		newData[key] = []byte(value)
	}

	return CreateSecret(client, cloudProvider, namespace, newData)
}

func CreateSecret(kc kubernetes.Interface, name, namespace string, data map[string][]byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		if _, err := kc.CoreV1().Secrets(namespace).Get(secret.Name, metav1.GetOptions{}); err == nil {
			log.Infof("Secret %q Already Exists, Ignoring", secret.Name)
			return true, nil
		}

		_, err := kc.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			log.Info(err)
		}
		return err == nil, nil
	})
}

func CreateNamespace(kc kubernetes.Interface, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
		},
	}
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		if _, err := kc.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{}); err == nil {
			log.Infof("Namespace %q Already Exists, Ignoring", ns.Name)
			return true, nil
		}

		_, err := kc.CoreV1().Namespaces().Create(ns)
		if err != nil {
			log.Info(err)
		}
		return err == nil, nil
	})
}

func CreateConfigMap(kc kubernetes.Interface, name, namespace string, data map[string]string) error {
	conf := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		_, err := kc.CoreV1().ConfigMaps(namespace).Create(conf)

		if err != nil {
			log.Info(err)
		}
		return err == nil, nil
	})
}

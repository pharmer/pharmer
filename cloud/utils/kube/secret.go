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
package kube

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/appscode/go/log"
	"github.com/appscode/go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

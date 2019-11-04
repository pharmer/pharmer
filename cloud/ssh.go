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
package cloud

import (
	"fmt"
	"net"
	"os"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExecuteTCPCommand(command, addr string, config *ssh.ClientConfig) (string, error) {
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	session.Stdout = DefaultWriter
	session.Stderr = DefaultWriter
	session.Stdin = os.Stdin
	if config.User != "root" {
		command = fmt.Sprintf("sudo %s", command)
	}
	_ = session.Run(command)
	output := DefaultWriter.Output()
	_ = session.Close()
	return output, nil
}

var DefaultWriter = &StringWriter{
	data: make([]byte, 0),
}

type StringWriter struct {
	data []byte
}

func (s *StringWriter) Flush() {
	s.data = make([]byte, 0)
}

func (s *StringWriter) Output() string {
	return string(s.data)
}

func (s *StringWriter) Write(b []byte) (int, error) {
	log.Infoln("$ ", string(b))
	s.data = append(s.data, b...)
	return len(b), nil
}

func GetSSHConfig(storeProvider store.ResourceInterface, clusterName, nodeName string) (*api.SSHConfig, error) {
	cluster, err := storeProvider.Clusters().Get(clusterName)
	if err != nil {
		return nil, err
	}

	scope := NewScope(NewScopeParams{
		Cluster:       cluster,
		StoreProvider: storeProvider,
	})
	cm, err := scope.GetCloudManager()
	if err != nil {
		return nil, err
	}

	client, err := cm.GetAdminClient()
	if err != nil {
		return nil, err
	}

	node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	term.ExitOnError(err)

	cfg := &api.SSHConfig{
		PrivateKey: scope.Certs.SSHKey.PrivateKey,
		User:       cluster.Spec.Config.SSHUserName,
		HostPort:   int32(22),
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			cfg.HostIP = addr.Address
		}
	}
	if net.ParseIP(cfg.HostIP) == nil {
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}

	return cfg, nil
}

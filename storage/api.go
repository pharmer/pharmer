package storage

import "github.com/appscode/pharmer/api"

type Storage interface {
	KubernetesStore
	KubernetesInstanceStore
}

type KubernetesStore interface {
	GetActiveCluster(name string) ([]*api.Kubernetes, error)
}

type KubernetesInstanceStore interface {
}

type CertificateStore interface {
}

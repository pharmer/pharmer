package storage

type Storage interface {
	KubernetesStore
	KubernetesInstanceStore
}

type KubernetesStore interface {
	GetActiveCluster(name string) ([]*Kubernetes, error)
}

type KubernetesInstanceStore interface {
}

type CertificateStore interface {
}

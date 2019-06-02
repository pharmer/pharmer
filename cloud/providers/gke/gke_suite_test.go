package gke_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGke(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gke Suite")
}

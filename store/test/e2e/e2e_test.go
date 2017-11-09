package e2e

import (
	"testing"
	_ "github.com/onsi/gomega"
	_ "github.com/onsi/ginkgo"
)

func TestE2E(t *testing.T) {
	RunE2ETestSuit(t)
}

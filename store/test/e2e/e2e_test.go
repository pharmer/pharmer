package e2e

import (
	"testing"
	_ "github.com/onsi/ginkgo"
	_ "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RunE2ETestSuit(t)
}

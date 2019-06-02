package machinesetup_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMachinesetup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machinesetup Suite")
}

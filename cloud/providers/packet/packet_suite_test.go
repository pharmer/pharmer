package packet_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPacket(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Packet Suite")
}

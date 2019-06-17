package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pharmer/pharmer/test/e2e/util"
)

var _ = FDescribe("test create cluster command", func() {
	type cases struct {
		args string
	}

	AfterEach(func() {
		// delete credential
		// delete cluster
	})

	table := func() interface{} {
		return DescribeTable("create cluster scenerios",
			func(c cases) {
				err, _ := util.RunCommand(strings.Split(c.args, " "))
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("gce", cases{
				args: "pharmer create cluster gce-1 --masters 3 --provider gce --zone us-central1-f --nodes n1-standard-2=3 --credential-uid google --kubernetes-version 1.14.0",
			}),
			Entry("aws", cases{
				args: "pharmer create cluster aws-1 --masters 3 --provider aws --zone us-east-1b --nodes t2.medium=1 --credential-uid aws --kubernetes-version v1.13.5",
			}),
		)
	}

	Context("for local provider", func() {
		// set pharmer context

		// running tests
		table()
	})

	Context("for xorm provider", func() {
		table()
	})

	Context("for fake provider", func() {
		table()
	})
})

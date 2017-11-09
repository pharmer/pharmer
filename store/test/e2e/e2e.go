package e2e

import (
	"flag"
	"path/filepath"
	"testing"
	"time"

	"fmt"

	"github.com/appscode/pharmer/store/test/framework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/util/homedir"
	//logs "github.com/appscode/log/golog"
)

const (
	TestTimeout = 2 * time.Hour
)

var (
	root       *framework.Framework
//	invocation *framework.Invocation
	configFile string
)

func init() {
	flag.StringVar(&configFile, "config-file", filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default"), "Storage provider configuration file")
}

func RunE2ETestSuit(t *testing.T) {
	//logs.InitLogs()
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TestTimeout)

	junitReporter := reporters.NewJUnitReporter("report.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Storage E2E Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	root = framework.New(configFile)
	fmt.Println(root.Config.GetStoreType())
	By("Using storage provider " + root.Config.GetStoreType())
})

var _ = AfterSuite(func() {
	By("Deleting credential")
	err :=root.Storage.Credentials().Delete(root.Invoke().Credential.GetName())
	Expect(err).NotTo(HaveOccurred())

	By("Deleting cluster")
	err = root.Storage.Clusters().Delete(root.Invoke().Cluster.GetName())
	Expect(err).NotTo(HaveOccurred())

})

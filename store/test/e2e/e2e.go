package e2e

import (
	"flag"
	"fmt"
	"path/filepath"
	"testing"
	"time"

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
	root *framework.Framework
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
	root.ClusterName = "storage-test"
	fmt.Println(root.Config.GetStoreType())
	By("Using storage provider " + root.Config.GetStoreType())
})

var _ = AfterSuite(func() {

})

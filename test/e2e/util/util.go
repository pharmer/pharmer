package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pharmer/cloud/pkg/apis"

	"flag"

	"github.com/onsi/gomega/gexec"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	ClusterName    string
	CredentialName string
	Providers      string
	pathToCredData string

	Masters           int
	CurrentVersion    string
	UpdateToVersion   string
	SkipDeleteCluster bool

	pharmerPath string
)

var defaultZone = map[string]string{
	apis.AWS:          "us-east-1b",
	apis.Azure:        "eastus2",
	apis.GCE:          "us-central1-f",
	apis.DigitalOcean: "nyc1",
	apis.Linode:       "us-central",
	apis.Packet:       "ewr1",
}
var defaultNodes = map[string]string{
	apis.AWS:          "t2.medium",
	apis.Azure:        "Standard_B2ms",
	apis.GCE:          "n1-standard-2",
	apis.DigitalOcean: "2gb",
	apis.Linode:       "g6-standard-2",
	apis.Packet:       "baremetal_0",
}

func init() {
	flag.StringVar(&Providers, "providers", "", "comma seperated provider names")
	flag.StringVar(&pathToCredData, "from-file", "", "File path for credential")

	flag.StringVar(&CurrentVersion, "current-version", "1.13.5", "Kubernetes version to be created")
	flag.StringVar(&UpdateToVersion, "update-to", "1.14.0", "Kubernetes version to be upgraded")

	flag.IntVar(&Masters, "masters", 1, "Number of masters")
	flag.BoolVar(&SkipDeleteCluster, "skip-delete", false, "Skip delete ClusterName")

	flag.Parse()
}

func SetClusterName() {
	ClusterName = "pharmer-test-" + rand.Characters(6)
	CredentialName = fmt.Sprintf("%s-credential", ClusterName)
}

func getRestConfig() *rest.Config {
	By("Getting rest config")
	return config.GetConfigOrDie()
}

func WaitForNodeReady(role string, numNodes int) {
	kc, err := kubernetes.NewForConfig(getRestConfig())
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("Waiting for %s to be ready", role))

	count := 1
	err = wait.Poll(5*time.Second, 30*time.Minute, func() (bool, error) {
		fmt.Println("Attempt", count, ": Waiting for the Nodes to be Ready . . . .")
		nodeList, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			log.Infof("failed to list nodes: %v", err)
			return false, nil
		}
		numReadyNodes := 0

		for _, node := range nodeList.Items {
			for _, taint := range node.Spec.Taints {
				if taint.Key == "node.cloudprovider.kubernetes.io/uninitialized" {
					log.Infof("Node %s not ready", node.Name)
					continue
				}
			}

			_, ok := node.Labels["node-role.kubernetes.io/master"]
			if ok {
				if role == "master" {
					numReadyNodes++
				}
				continue
			} else if role == "master" {
				continue
			}

			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					numReadyNodes++
				}
			}
		}

		log.Infof("Expected %d, Found %d", numNodes, numReadyNodes)

		count++
		return numReadyNodes == numNodes, nil

	})
	Expect(err).NotTo(HaveOccurred())
}

func ClusterApiClient() (clientset.Interface, error) {
	return clientset.NewForConfig(getRestConfig())
}

var BuildPharmer = func() {
	By("Building pharmer")
	var err error
	pharmerPath, err = gexec.Build("github.com/pharmer/pharmer")
	Expect(err).NotTo(HaveOccurred())
}

var CreateCredential = func() {
	for _, provider := range strings.Split(Providers, ",") {
		command := []string{
			"pharmer", "create", "credential", CredentialName + "_" + provider,
			"--provider", provider,
		}

		if provider == "gce" {
			command = append(command, "--from-file", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
		} else {
			command = append(command, "--from-env")
		}

		runCommand(command)
	}
}

var DeleteCredential = func() {
	for _, provider := range strings.Split(Providers, ",") {
		command := []string{
			"pharmer", "delete", "credential", CredentialName + "_" + provider,
		}

		runCommand(command)
	}
}

var CreateCluster = func(provider, version string) {
	command := []string{
		"pharmer", "create", "cluster", ClusterName,
		"--provider", provider,
		"--masters", fmt.Sprintf("%d", Masters),
		"--zone", defaultZone[provider],
		"--nodes", defaultNodes[provider] + "=1",
		"--credential-uid", CredentialName + "_" + provider,
		"--kubernetes-version", version,
	}

	runCommand(command)
}

var ApplyCluster = func() {
	command := []string{"pharmer", "apply", ClusterName}
	runCommand(command)
}

var UseCluster = func() {
	command := []string{"pharmer", "use", "cluster", ClusterName}
	runCommand(command)
}

var DeleteCluster = func() {
	command := []string{"pharmer", "delete", "cluster", ClusterName}
	runCommand(command)
}

var ScaleCluster = func(n int32) {
	By("Getting Cluster API client")
	caClient, err := ClusterApiClient()
	Expect(err).NotTo(HaveOccurred())

	By("Getting MachineSet")
	machineSets, err := caClient.ClusterV1alpha1().MachineSets(metav1.NamespaceDefault).List(metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	By("Updating Machines")
	for _, machineSet := range machineSets.Items {
		machineSet.Spec.Replicas = &n
		_, err = caClient.ClusterV1alpha1().MachineSets(metav1.NamespaceDefault).Update(&machineSet)
		Expect(err).NotTo(HaveOccurred())
	}

	By(fmt.Sprintf("Waiting for %v Nodes to become ready", n))

	WaitForNodeReady("node", int(n))
}

//var UpgradeCluster = func() {
//	By("Upgrading cluster")
//
//	command := []string{
//		"pharmer", "edit", "cluster", ClusterName,
//		"--kubernetes-version=", updateToVersion,
//	}
//
//	runCommand(command)
//}

//var WaitForUpdates = func() {
//	kc, err := kubernetes.NewForConfig(getRestConfig())
//	Expect(err).NotTo(HaveOccurred())
//
//	count := 1
//	err = wait.Poll(5*time.Second, 15*time.Minute, func() (bool, error) {
//		fmt.Println("Attempt", count, ": Waiting for the Nodes to be Ready . . . .")
//		nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
//		if err != nil {
//			log.Infof("failed to list nodes: %v", err)
//			return false, nil
//		}
//
//		for _, node := range nodes.Items {
//			if node.Status.NodeInfo.KubeletVersion != updateToVersion {
//				log.Infof("expected kubernetes version %s for node %s, found version %s",
//					updateToVersion, node.Name, node.Status.NodeInfo.KubeletVersion)
//				return false, nil
//			}
//		}
//		return true, nil
//	})
//
//	Expect(err).NotTo(HaveOccurred())
//}

func runCommand(command []string) {
	cmd := exec.Command(pharmerPath, command[1:]...)

	stderr, err := cmd.StderrPipe()
	Expect(err).NotTo(HaveOccurred())

	stdout, err := cmd.StdoutPipe()
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("Running Command: %v", command))

	err = cmd.Start()
	Expect(err).NotTo(HaveOccurred())

	go streamReader(stderr)
	go streamReader(stdout)

	err = cmd.Wait()
	Expect(err).NotTo(HaveOccurred())
}

func streamReader(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
}

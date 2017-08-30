package gce

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/mgutz/str"
	compute "google.golang.org/api/compute/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	//if !cloud.UpgradeRequired(cm.cluster, req) {
	//	cloud.Logger(cm.ctx).Infof("Upgrade command skipped for cluster %v", cm.cluster.Name)
	//	return nil // TODO check error nil
	//}

	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}

	cm.cluster.Spec.ResourceVersion = int64(0)
	cm.namer = namer{cluster: cm.cluster}
	cm.updateContext()
	// assign new timestamp and new launch_config version
	cm.cluster.Spec.EnvTimestamp = time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	cm.cluster.Spec.KubernetesVersion = req.KubernetesVersion
	_, err := cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)

	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	fmt.Println("Updating...")
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, _ = cloud.Store(cm.ctx).Instances(cm.cluster.Name).List(api.ListOptions{})
	if req.ApplyToMaster {
		for _, instance := range cm.ins.Instances {
			if instance.Spec.Role == api.RoleKubernetesMaster {
				cm.masterUpdate(instance.Status.ExternalIP, instance.Name, req.KubernetesVersion)
			}
		}
		//err = cm.updateMaster()
		//if err != nil {
		//	return errors.FromErr(err).WithContext(cm.ctx).Err()
		//}
	} else {
		err = cm.updateNodes(req.Sku)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	_, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) masterUpdate(host, instanceName, version string) error {
	/*fmt.Println(string(cm.ctx.SSHKey.PrivateKey))
	signer, err := ssh.MakePrivateKeySignerFromBytes(cm.ctx.SSHKey.PrivateKey)
	if err != nil {
		fmt.Println(err)
	}
	sout, serr, code, err := ssh.Exec("ls -l /", "", host+":22", signer)
	fmt.Println("------------------------")
	fmt.Println(sout, serr, code, err)
	fmt.Println("------------------------")*/
	command := fmt.Sprintf(`gcloud compute --project "%v" ssh --zone "%v" "%v"`, cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instanceName)
	init := fmt.Sprintf(`sudo kubeadm init --apiserver-bind-port 6443 --token %v  --apiserver-advertise-address ${PUBLICIP} --apiserver-cert-extra-sans ${PUBLICIP} ${PRIVATEIP} --pod-network-cidr 10.244.0.0/16 --kubernetes-version %v --skip-preflight-checks`,
		cm.cluster.Spec.KubeadmToken, "v"+version)
	fmt.Println(init)
	arg := str.ToArgv(command)
	name, arg := arg[0], arg[1:]
	//arg = append(arg, "--command", "ls -lah")
	cmd := exec.Command(name, arg...)
	stdIn := newStringReader([]string{
		"ls -lah",
		"sudo apt-get update",
		"sudo apt-get upgrade",
		"sudo systemctl restart kubelet",
		"sudo KUBECONFIG=/etc/kubernetes/admin.conf kubectl delete daemonset kube-proxy -n kube-system",
		"PUBLICIP=$(curl ipinfo.io/ip)",
		"PRIVATEIP=$(ip route get 8.8.8.8 | awk '{print $NF; exit}')",
		init,
	})
	cmd.Stdin = stdIn
	cmd.Stdout = DefaultWriter
	cmd.Stderr = DefaultWriter
	err := cmd.Run()
	output := DefaultWriter.Output()
	fmt.Println(output, err)
	if err := cloud.WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	_, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	/*keySigner, _ := ssh.ParsePrivateKey(cm.ctx.SSHKey.PrivateKey)
	config := &ssh.ClientConfig{
		User: "sanjid",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keySigner),
		},
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%v:%v", host, 22), config)
	fmt.Println(err)
	defer conn.Close()
	session, _ := conn.NewSession()
	session.Stdout = DefaultWriter
	session.Stderr = DefaultWriter
	session.Stdin = os.Stdin
	session.Run("ls -lah")
	output := DefaultWriter.Output()
	session.Close()
	fmt.Println(output)*/
	return nil
}

var DefaultWriter = &StringWriter{
	data: make([]byte, 0),
}

type StringWriter struct {
	data []byte
}

func (s *StringWriter) Flush() {
	s.data = make([]byte, 0)
}

func (s *StringWriter) Output() string {
	return string(s.data)
}

func (s *StringWriter) Write(b []byte) (int, error) {
	//log.Infoln("$ ", string(b))
	s.data = append(s.data, b...)
	return len(b), nil
}

func newStringReader(ss []string) io.Reader {
	formattedString := strings.Join(ss, "\n")
	reader := strings.NewReader(formattedString)
	return reader
}

/*
func (cm *clusterManager) updateMaster() error {
	fmt.Println("Updating Master...")
	err := cm.deleteMaster()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	img, err := cm.conn.getInstanceImage()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Spec.InstanceImage = img
	cm.UploadStartupConfig()

	op, err := cm.createMasterIntance()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForZoneOperation(op)

	if err := cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance, err := cm.getInstance(cm.cluster.Spec.KubernetesMasterName)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// cm.ins.Instances = nil
	// cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	fmt.Println(masterInstance)
	fmt.Println("Master Updated")
	return nil
}
*/

func (cm *ClusterManager) nodeUpdate(instanceName string) error {
	command := fmt.Sprintf(`gcloud compute --project "%v" ssh --zone "%v" "%v"`, cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instanceName)
	arg := str.ToArgv(command)
	name, arg := arg[0], arg[1:]
	//arg = append(arg, "--command", "ls -lah")
	cmd := exec.Command(name, arg...)
	stdIn := newStringReader([]string{
		"ls -lah",
		"sudo apt-get update",
		"sudo apt-get upgrade",
		"sudo systemctl restart kubelet",
	})
	cmd.Stdin = stdIn
	cmd.Stdout = DefaultWriter
	cmd.Stderr = DefaultWriter
	err := cmd.Run()
	output := DefaultWriter.Output()
	fmt.Println(output, err)
	if err := cloud.WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	_, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")

	newInstanceTemplate := cm.namer.InstanceTemplateName(sku)
	op, err := cm.createNodeInstanceTemplate(sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForGlobalOperation(op)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	ctxV, err := cloud.GetExistingContextVersion(cm.ctx, cm.cluster, sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	oldInstanceTemplate := cm.namer.InstanceTemplateNameWithContext(sku, ctxV)
	groupName := cm.namer.InstanceGroupName(sku)
	oldinstances, err := cm.listInstances(groupName)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances := []string{}
	prefix := "https://www.googleapis.com/compute/v1/projects/" + cm.cluster.Spec.Project + "/zones/" + cm.cluster.Spec.Zone + "/instances/"
	for _, instance := range oldinstances {
		instanceName := prefix + instance.Name
		instances = append(instances, instanceName)
	}
	err = cm.rollingUpdate(instances, newInstanceTemplate, sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	currentIns, err := cm.listInstances(groupName)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cloud.AdjustDbInstance(cm.ctx, cm.ins, currentIns, sku)
	// cluster.Spec.ctx.Instances = append(cluster.Spec.ctx.Instances, instances...)
	_, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cm.deleteInstanceTemplate(oldInstanceTemplate)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	return nil
}

func (cm *ClusterManager) getExistingContextVersion(sku string) (error, int64) {
	kc, err := cloud.NewAdminClient(cm.ctx, cm.cluster)
	if err != nil {
		log.Fatal(err)
	}
	//re, _ := labels.NewRequirement(api.NodeLabelKey_SKU, selection.Equals, []string{sku})
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{
	//LabelSelector: labels.Selector.Add(*re).Matches(labels.Labels(api.NodeLabelKey_SKU)),
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, n := range nodes.Items {
		nl := api.FromMap(n.GetLabels())
		if nl.GetString(api.NodeLabelKey_SKU) == sku {
			return nil, nl.GetInt64(api.NodeLabelKey_ContextVersion)
		}
	}
	return errors.New("Context version not found").Err(), int64(0)
}

func (cm *ClusterManager) rollingUpdate(oldInstances []string, newInstanceTemplate, sku string) error {
	groupName := cm.namer.InstanceGroupName(sku)
	newTemplate := fmt.Sprintf("projects/%v/global/instanceTemplates/%v", cm.cluster.Spec.Project, newInstanceTemplate)
	template := &compute.InstanceGroupManagersSetInstanceTemplateRequest{
		InstanceTemplate: newTemplate,
	}
	tmpR, err := cm.conn.computeService.InstanceGroupManagers.SetInstanceTemplate(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, groupName, template).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn.waitForZoneOperation(tmpR.Name)
	fmt.Println("rolling update started...")

	// gcloud compute --project "tigerworks-kube" instance-groups managed recreate-instances  "gce153-n1-standard-2-v160" --zone "us-central1-f" --instances "gce153-node-n1-standard-2-whf8"
	for _, instance := range oldInstances {
		fmt.Println("updating ", instance)
		updates := &compute.InstanceGroupManagersRecreateInstancesRequest{
			Instances: []string{instance},
		}
		r, err := cm.conn.computeService.InstanceGroupManagers.RecreateInstances(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, groupName, updates).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		cm.conn.waitForZoneOperation(r.Name)
		fmt.Println("Waiting for 1 minute")
		time.Sleep(1 * time.Minute)
	}

	return nil
}

package linode

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/appscode/go/types"
	"github.com/linode/linodego"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *linodego.Client
	namer   namer
}

func NewConnector(ctx context.Context, cluster *api.Cluster, owner string) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Linode{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cluster.ClusterConfig().CredentialName, err)
	}

	namer := namer{cluster: cluster}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: typed.APIToken()})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	c := linodego.NewClient(oauth2Client)

	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer,
		client:  &c,
	}, nil
}

func PrepareCloud(ctx context.Context, clusterName string, owner string) (*cloudConnector, error) {
	var err error
	var conn *cloudConnector
	cluster, err := Store(ctx).Owner(owner).Clusters().Get(clusterName)
	if err != nil {
		return conn, fmt.Errorf("cluster `%s` does not exist. Reason: %v", clusterName, err)
	}

	if ctx, err = LoadCACertificates(ctx, cluster, owner); err != nil {
		return conn, err
	}

	if ctx, err = LoadSSHKey(ctx, cluster, owner); err != nil {
		return conn, err
	}
	if conn, err = NewConnector(ctx, cluster, owner); err != nil {
		return nil, err
	}
	return conn, nil
}

func (conn *cloudConnector) DetectInstanceImage() (string, error) {
	filter := `{"label":"Ubuntu 16.04 LTS"}`
	listOpts := &linodego.ListOptions{nil, filter}

	images, err := conn.client.ListImages(context.Background(), listOpts)
	if err != nil {
		return "", err
	}

	if len(images) == 0 {
		return "", errors.New("image Ubuntu 16.04 LTS not found")
	} else if len(images) > 1 {
		return "", errors.New("multiple images found")
	}

	return images[0].ID, nil
}

func (conn *cloudConnector) DetectKernel() (string, error) {
	kernels, err := conn.client.ListKernels(conn.ctx, nil)
	if err != nil {
		return "", err
	}
	kernelId := ""
	for _, d := range kernels {
		if d.PVOPS {
			if strings.HasPrefix(d.Label, "Latest 64 bit") {
				return d.ID, nil
			}
			if strings.Contains(d.Label, "x86_64") && d.ID != kernelId {
				kernelId = d.ID
			}
		}
	}
	if kernelId != "" {
		return kernelId, nil
	}
	return "", errors.New("can't find Kernel")
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) waitForStatus(id int, status linodego.InstanceStatus) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		instance, err := conn.client.GetInstance(context.Background(), id)
		if err != nil {
			return false, nil
		}
		if instance == nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%v`", attempt, instance.Label, instance.Status)
		if instance.Status == status {
			return true, nil
		}
		return false, nil
	})
}

/*
Status values are -1: Being Created, 0: Brand New, 1: Running, and 2: Powered Off.
*/
func statusString(status int) string {
	switch status {
	case LinodeStatus_BeingCreated:
		return "Being Created"
	case LinodeStatus_BrandNew:
		return "Brand New"
	case LinodeStatus_Running:
		return "Running"
	case LinodeStatus_PoweredOff:
		return "Powered Off"
	default:
		return strconv.Itoa(status)
	}
}

const (
	LinodeStatus_BeingCreated = -1
	LinodeStatus_BrandNew     = 0
	LinodeStatus_Running      = 1
	LinodeStatus_PoweredOff   = 2
)

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getStartupScriptID(machine *clusterv1.Machine) (int, error) {
	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return 0, err
	}
	fmt.Println("roles = " + machineConfig.Roles[0])
	scriptName := conn.namer.StartupScriptName(machine.Name, string(machineConfig.Roles[0]))
	filter := fmt.Sprintf(`{"label" : "%v"}`, scriptName)
	listOpts := &linodego.ListOptions{nil, filter}

	scripts, err := conn.client.ListStackscripts(context.Background(), listOpts)
	if err != nil {
		return 0, err
	}

	if len(scripts) > 1 {
		return 0, errors.Errorf("multiple stackscript found with label %v", scriptName)
	} else if len(scripts) == 0 {
		return 0, errors.Errorf("no stackscript found with label %v", scriptName)
	}
	return scripts[0].ID, nil
}

func (conn *cloudConnector) createOrUpdateStackScript(cluster *api.Cluster, machine *clusterv1.Machine, token string, owner string) (int, error) {
	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return 0, err
	}
	scriptName := conn.namer.StartupScriptName(machine.Name, string(machineConfig.Roles[0]))
	script, err := conn.renderStartupScript(cluster, machine, token, owner)
	if err != nil {
		return 0, err
	}

	filter := fmt.Sprintf(`{"label" : "%v"}`, scriptName)
	listOpts := &linodego.ListOptions{nil, filter}

	scripts, err := conn.client.ListStackscripts(context.Background(), listOpts)
	if err != nil {
		return 0, err
	}

	if len(scripts) > 1 {
		return 0, errors.Errorf("multiple stackscript found with label %v", scriptName)
	} else if len(scripts) == 0 {
		createOpts := linodego.StackscriptCreateOptions{
			Label:       scriptName,
			Description: fmt.Sprintf("Startup script for NodeGroup %s of Cluster %s", machine.Name, conn.cluster.Name),
			Images:      []string{conn.cluster.ClusterConfig().Cloud.InstanceImage},
			Script:      script,
		}
		stackScript, err := conn.client.CreateStackscript(context.Background(), createOpts)
		if err != nil {
			return 0, err
		}
		Logger(conn.ctx).Infof("Stack script for role %v created", string(machineConfig.Roles[0]))
		return stackScript.ID, nil
	}

	updateOpts := scripts[0].GetUpdateOptions()
	updateOpts.Script = script

	stackScript, err := conn.client.UpdateStackscript(context.Background(), scripts[0].ID, updateOpts)
	if err != nil {
		return 0, err
	}

	Logger(conn.ctx).Infof("Stack script for role %v updated", string(machineConfig.Roles[0]))
	return stackScript.ID, nil
}

func (conn *cloudConnector) DeleteStackScript(machineName string, role string) error {
	scriptName := conn.namer.StartupScriptName(machineName, role)
	filter := fmt.Sprintf(`{"label" : "%v"}`, scriptName)
	listOpts := &linodego.ListOptions{nil, filter}

	scripts, err := conn.client.ListStackscripts(context.Background(), listOpts)
	if err != nil {
		return err
	}
	for _, script := range scripts {
		if err := conn.client.DeleteStackscript(context.Background(), script.ID); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------
func (conn *cloudConnector) CreateInstance(name, token string, machine *clusterv1.Machine, owner string) (*api.NodeInfo, error) {
	if _, err := conn.createOrUpdateStackScript(conn.cluster, machine, token, owner); err != nil {
		return nil, err
	}
	scriptId, err := conn.getStartupScriptID(machine)
	if err != nil {
		return nil, err
	}
	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	createOpts := linodego.InstanceCreateOptions{
		Label:    name,
		Region:   conn.cluster.ClusterConfig().Cloud.Zone,
		Type:     machineConfig.Type,
		RootPass: conn.cluster.ClusterConfig().Cloud.Linode.RootPassword,
		AuthorizedKeys: []string{
			string(SSHKey(conn.ctx).PublicKey),
		},
		StackScriptData: map[string]string{
			"hostname": name,
		},
		StackScriptID:  scriptId,
		Image:          conn.cluster.ClusterConfig().Cloud.InstanceImage,
		BackupsEnabled: false,
		PrivateIP:      true,
		SwapSize:       types.IntP(0),
	}

	instance, err := conn.client.CreateInstance(context.Background(), createOpts)
	if err != nil {
		return nil, err
	}

	if err := conn.waitForStatus(instance.ID, linodego.InstanceRunning); err != nil {
		return nil, err
	}

	node := api.NodeInfo{
		Name:       instance.Label,
		ExternalID: strconv.Itoa(instance.ID),
		PublicIP:   instance.IPv4[0].String(),
		PrivateIP:  instance.IPv4[1].String(),
	}
	return &node, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	id, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return err
	}

	if err := conn.client.DeleteInstance(context.Background(), id); err != nil {
		return err
	}

	Logger(conn.ctx).Infof("Instance %v deleted", id)
	return nil
}

func instanceIDFromProviderID(providerID string) (int, error) {
	if providerID == "" {
		return 0, errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return 0, errors.Errorf("unexpected providerID format: %s, format should be: linode://12345", providerID)
	}

	// since split[0] is actually "linode:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return 0, errors.Errorf("provider name from providerID should be linode: %s", providerID)
	}

	return strconv.Atoi(split[2])
}

func (conn *cloudConnector) instanceIfExists(machine *clusterv1.Machine) (*linodego.Instance, error) {
	lds, err := conn.client.ListInstances(conn.ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, ld := range lds {
		if ld.Label == machine.Name {
			return &ld, nil
		}
	}

	return nil, fmt.Errorf("no droplet found with %v name", machine.Name)
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	cred := make(map[string]string)
	cred["apiToken"] = data["token"]
	cred["region"] = conn.cluster.ClusterConfig().Cloud.Region
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ccm-" + conn.cluster.ClusterConfig().Cloud.CloudProvider,
		},
		StringData: cred,
		Type:       core.SecretTypeOpaque,
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := kc.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret)
		return err == nil, nil
	})
}

// ---------------------------------------------------------------------------------------------------------------------

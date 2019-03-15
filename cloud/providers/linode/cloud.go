package linode

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/appscode/go/types"
	"github.com/linode/linodego"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *linodego.Client
	namer   namer
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Linode{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	namer := namer{cluster: cluster}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: typed.APIToken()})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	linodeClient := linodego.NewClient(oauth2Client)

	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer,
		client:  &linodeClient,
	}, nil
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

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getStartupScriptID(ng *api.NodeGroup) (int, error) {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())

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

func (conn *cloudConnector) createOrUpdateStackScript(ng *api.NodeGroup, token string) (int, error) {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())
	script, err := conn.renderStartupScript(ng, token)
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
			Description: fmt.Sprintf("Startup script for NodeGroup %s of Cluster %s", ng.Name, conn.cluster.Name),
			Images:      []string{conn.cluster.Spec.Cloud.InstanceImage},
			Script:      script,
		}
		stackScript, err := conn.client.CreateStackscript(context.Background(), createOpts)
		if err != nil {
			return 0, err
		}
		Logger(conn.ctx).Infof("Stack script for role %v created", ng.Role())
		return stackScript.ID, nil
	}

	updateOpts := scripts[0].GetUpdateOptions()
	updateOpts.Script = script

	stackScript, err := conn.client.UpdateStackscript(context.Background(), scripts[0].ID, updateOpts)
	if err != nil {
		return 0, err
	}

	Logger(conn.ctx).Infof("Stack script for role %v updated", ng.Role())
	return stackScript.ID, nil
}

func (conn *cloudConnector) deleteStackScript(ng *api.NodeGroup) error {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())

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

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	scriptId, err := conn.getStartupScriptID(ng)
	if err != nil {
		return nil, err
	}

	createOpts := linodego.InstanceCreateOptions{
		Label:    name,
		Region:   conn.cluster.Spec.Cloud.Zone,
		Type:     ng.Spec.Template.Spec.SKU,
		RootPass: conn.cluster.Spec.Cloud.Linode.RootPassword,
		AuthorizedKeys: []string{
			string(SSHKey(conn.ctx).PublicKey),
		},

		StackScriptID: scriptId,
		StackScriptData: map[string]string{
			"hostname": name,
		},
		Image:          conn.cluster.Spec.Cloud.InstanceImage,
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

/*
func (conn *cloudConnector) bootToGrub2(linodeId, configId int, name string) error {
	// GRUB2 Kernel ID = 210
	_, err := conn.client.Config.Update(configId, linodeId, 210, nil)
	if err != nil {
		return err
	}
	_, err = conn.client.Linode.Update(linodeId, map[string]interface{}{
		"Label":    name,
		"watchdog": true,
	})
	if err != nil {
		return err
	}
	_, err = conn.client.Linode.Boot(linodeId, configId)
	Logger(conn.ctx).Infof("%v booted", name)
	return err
}
*/

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	id, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}

	if err := conn.client.DeleteInstance(context.Background(), id); err != nil {
		return err
	}

	Logger(conn.ctx).Infof("Instance %v deleted", id)
	return nil
}

// dropletIDFromProviderID returns a droplet's ID from providerID.
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: linode://droplet-id
// ref: https://github.com/digitalocean/digitalocean-cloud-controller-manager/blob/f9a9856e99c9d382db3777d678f29d85dea25e91/do/droplets.go#L211
func serverIDFromProviderID(providerID string) (int, error) {
	if providerID == "" {
		return 0, errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return 0, errors.Errorf("unexpected providerID format: %s, format should be: linode://12345", providerID)
	}

	// since split[0] is actually "linode:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return 0, errors.Errorf("provider name from providerID should be %s: %s", providerID, UID)
	}

	return strconv.Atoi(split[2])
}

// ---------------------------------------------------------------------------------------------------------------------

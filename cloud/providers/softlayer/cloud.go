package softlayer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/data"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx                   context.Context
	cluster               *api.Cluster
	virtualServiceClient  services.Virtual_Guest
	accountServiceClient  services.Account
	securityServiceClient services.Security_Ssh_Key
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Softlayer{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	sess := session.New(typed.Username(), typed.APIKey())
	sess.Debug = true
	return &cloudConnector{
		ctx:                   ctx,
		cluster:               cluster,
		virtualServiceClient:  services.GetVirtualGuestService(sess),
		accountServiceClient:  services.GetAccountService(sess),
		securityServiceClient: services.GetSecuritySshKeyService(sess),
	}, nil
}

func (conn *cloudConnector) waitForInstance(id int) {
	service := conn.virtualServiceClient.Id(id)

	// Delay to allow transactions to be registered
	for transactions, _ := service.GetActiveTransactions(); len(transactions) > 0; {
		fmt.Print(".")
		time.Sleep(30 * time.Second)
		transactions, _ = service.GetActiveTransactions()
	}
	for yes, _ := service.IsPingable(); !yes; {
		fmt.Print(".")
		time.Sleep(15 * time.Second)
		yes, _ = service.IsPingable()
	}
	for yes, _ := service.IsBackendPingable(); !yes; {
		fmt.Print(".")
		time.Sleep(15 * time.Second)
		yes, _ = service.IsBackendPingable()
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, string, error) {
	return false, "", nil
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Debugln("Adding SSH public key")

	securitySSHTemplate := datatypes.Security_Ssh_Key{
		Label: StringP(conn.cluster.Name),
		Key:   StringP(string(SSHKey(conn.ctx).PublicKey)),
	}
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		sk, e2 := conn.securityServiceClient.CreateObject(&securitySSHTemplate)
		if e2 != nil {
			return false, nil
		}
		conn.cluster.Status.SSHKeyExternalID = strconv.Itoa(*sk.Id)
		return true, nil
	})
	if err != nil {
		return nil
	}
	Logger(conn.ctx).Debugf("Created new ssh key with fingerprint=%v", SSHKey(conn.ctx).OpensshFingerprint)
	return nil
}

func (conn *cloudConnector) deleteSSHKey(id int) error {
	Logger(conn.ctx).Infof("Deleting SSH key for cluster", conn.cluster.Name)
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		// id, _ := strconv.Atoi(conn.cluster.Status.SSHKeyExternalID)
		_, e2 := conn.securityServiceClient.Id(id).DeleteObject()
		return e2 == nil, nil
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v deleted", conn.cluster.Name)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.SimpleNode, error) {
	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return nil, err
	}

	instance, err := data.ClusterMachineType(conn.cluster.Spec.Cloud.CloudProvider, ng.Spec.Template.Spec.SKU)
	if err != nil {
		return nil, err
	}
	cpu := instance.CPU
	ram := 0
	switch instance.RAM.(type) {
	case int, int32, int64:
		ram = instance.RAM.(int) * 1024
	case float64, float32:
		ram = int(instance.RAM.(float64) * 1024)
	default:
		return nil, fmt.Errorf("failed to parse memory metadata for sku %v", ng.Spec.Template.Spec.SKU)
	}

	sshid, err := strconv.Atoi(conn.cluster.Status.SSHKeyExternalID)
	if err != nil {
		return nil, err
	}
	vGuestTemplate := datatypes.Virtual_Guest{
		Hostname:                     StringP(name),
		Domain:                       StringP(Extra(conn.ctx).ExternalDomain(conn.cluster.Name)),
		MaxMemory:                    IntP(ram),
		StartCpus:                    IntP(cpu),
		Datacenter:                   &datatypes.Location{Name: StringP(conn.cluster.Spec.Cloud.Zone)},
		OperatingSystemReferenceCode: StringP(conn.cluster.Spec.Cloud.OS),
		LocalDiskFlag:                TrueP(),
		HourlyBillingFlag:            TrueP(),
		SshKeys: []datatypes.Security_Ssh_Key{
			{
				Id:          IntP(sshid),
				Fingerprint: StringP(SSHKey(conn.ctx).OpensshFingerprint),
			},
		},
		UserData: []datatypes.Virtual_Guest_Attribute{
			{
				//https://sldn.softlayer.com/blog/jarteche/getting-started-user-data-and-post-provisioning-scripts
				Type: &datatypes.Virtual_Guest_Attribute_Type{
					Keyname: StringP("USER_DATA"),
					Name:    StringP("User Data"),
				},
				Value: StringP(script),
			},
		},
		PostInstallScriptUri: StringP("https://raw.githubusercontent.com/appscode/pharmer/master/cloud/providers/softlayer/startupscript.sh"),
	}

	vGuest, err := conn.virtualServiceClient.Mask("id;domain").CreateObject(&vGuestTemplate)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Softlayer instance %v created", name)

	serverID := *vGuest.Id

	// record nodes
	conn.waitForInstance(serverID)

	bluemix := conn.virtualServiceClient.Id(serverID)
	host, err := bluemix.GetObject()
	if err != nil {
		return nil, err
	}

	node := api.SimpleNode{
		Name:       *host.FullyQualifiedDomainName,
		ExternalID: strconv.Itoa(serverID),
	}
	node.PublicIP, err = bluemix.GetPrimaryIpAddress()
	if err != nil {
		return nil, err
	}
	node.PublicIP = strings.Trim(node.PublicIP, `"`)
	node.PrivateIP, err = bluemix.GetPrimaryBackendIpAddress()
	if err != nil {
		return nil, err
	}
	node.PrivateIP = strings.Trim(node.PrivateIP, `"`)

	return &node, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	id, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}
	success, err := conn.virtualServiceClient.Id(id).DeleteObject()
	if err != nil {
		return errors.FromErr(err).Err()
	} else if !success {
		return errors.New("Error deleting virtual guest").Err()
	}
	Logger(conn.ctx).Infof("Droplet %v deleted", id)
	return nil
}

// dropletIDFromProviderID returns a droplet's ID from providerID.
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: digitalocean://droplet-id
// ref: https://github.com/digitalocean/digitalocean-cloud-controller-manager/blob/f9a9856e99c9d382db3777d678f29d85dea25e91/do/droplets.go#L211
func serverIDFromProviderID(providerID string) (int, error) {
	if providerID == "" {
		return 0, errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return 0, fmt.Errorf("unexpected providerID format: %s, format should be: digitalocean://12345", providerID)
	}

	// since split[0] is actually "digitalocean:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return 0, fmt.Errorf("provider name from providerID should be digitalocean: %s", providerID)
	}

	return strconv.Atoi(split[2])
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) reboot(id int) (bool, error) {
	service := conn.virtualServiceClient.Id(id)
	return service.RebootDefault()
}

package softlayer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	. "github.com/appscode/go/types"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	"github.com/pharmer/cloud/pkg/providers"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx                   context.Context
	i                     providers.Interface
	cluster               *api.Cluster
	virtualServiceClient  services.Virtual_Guest
	accountServiceClient  services.Account
	securityServiceClient services.Security_Ssh_Key
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Softlayer{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.Config.CredentialName)
	}

	i, err := providers.NewCloudProvider(providers.Options{
		Provider: cluster.Spec.Config.Cloud.CloudProvider,
		// set credentials
	})
	if err != nil {
		return nil, err
	}

	sess := session.New(typed.Username(), typed.APIKey())
	sess.Debug = true
	return &cloudConnector{
		ctx:                   ctx,
		i:                     i,
		cluster:               cluster,
		virtualServiceClient:  services.GetVirtualGuestService(sess),
		accountServiceClient:  services.GetAccountService(sess),
		securityServiceClient: services.GetSecuritySshKeyService(sess),
	}, nil
}

func (conn *cloudConnector) waitForInstance(id int) error {
	service := conn.virtualServiceClient.Id(id)
	attempt := 0

	// wait for public IP
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		yes, e2 := service.IsPingable()
		if e2 != nil {
			Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is not pingable. Reason: `%s`", attempt, id, e2)
		}
		return yes, nil
	})
	if err != nil {
		return err
	}

	// wait for private IP
	err = wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		yes, e2 := service.IsBackendPingable()
		if e2 != nil {
			Logger(conn.ctx).Infof("Attempt %v: Instance `%v` backend is not pingable. Reason: `%s`", attempt, id, e2)
		}
		return yes, nil
	})
	if err != nil {
		return err
	}

	// wait for transactions to end
	err = wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		txs, e2 := service.GetActiveTransactions()
		if e2 != nil {
			Logger(conn.ctx).Infof("Attempt %v: Instance `%v` has pending transactions. Reason: `%s`", attempt, id, e2)
		}
		return len(txs) == 0, nil
	})
	return err
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, int, error) {
	if conn.cluster.Status.Cloud.SShKeyExternalID == "" {
		return false, -1, nil
	}
	sshKeys, err := conn.accountServiceClient.GetSshKeys()
	if err != nil {
		return false, -1, err
	}
	if id, err := strconv.Atoi(conn.cluster.Status.Cloud.SShKeyExternalID); err == nil {
		for _, sk := range sshKeys {
			if *sk.Id == id {
				return true, id, nil
			}
		}
	} else {
		for _, sk := range sshKeys {
			if *sk.Label == conn.cluster.Name {
				return true, *sk.Id, nil
			}
		}
	}
	return false, -1, nil
}

func (conn *cloudConnector) importPublicKey() (string, error) {
	Logger(conn.ctx).Debugln("Adding SSH public key")

	securitySSHTemplate := datatypes.Security_Ssh_Key{
		Label: StringP(conn.cluster.Name),
		Key:   StringP(string(SSHKey(conn.ctx).PublicKey)),
	}
	sk, err := conn.securityServiceClient.CreateObject(&securitySSHTemplate)
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Debugf("Created new ssh key with fingerprint=%v", SSHKey(conn.ctx).OpensshFingerprint)
	return strconv.Itoa(*sk.Id), nil
}

func (conn *cloudConnector) deleteSSHKey(id int) error {
	Logger(conn.ctx).Infof("Deleting SSH key for cluster %s", conn.cluster.Name)
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
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

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return nil, err
	}

	fmt.Println(script)

	mts, err := conn.i.ListMachineTypes()
	if err != nil {
		return nil, err
	}

	var instance *cloudapi.MachineType
	for _, x := range mts {
		if x.Spec.SKU == ng.Spec.Template.Spec.SKU {
			instance = &x
			break
		}
	}
	if instance == nil {
		return nil, errors.Errorf("can't find instance type %s for provider Packet", ng.Spec.Template.Spec.SKU)
	}

	// TODO: Fix this
	cpu, _ := instance.Spec.CPU.AsInt64()
	ram, _ := instance.Spec.RAM.AsInt64()

	_, sshid, err := conn.getPublicKey()
	if err != nil {
		return nil, err
	}
	domain := fmt.Sprintf("%v.pharmer.local", conn.cluster.Name)
	vGuestTemplate := datatypes.Virtual_Guest{
		Hostname:                     StringP(name),
		Domain:                       StringP(domain),
		MaxMemory:                    IntP(int(ram)),
		StartCpus:                    IntP(int(cpu)),
		Datacenter:                   &datatypes.Location{Name: StringP(conn.cluster.Spec.Config.Cloud.Zone)},
		OperatingSystemReferenceCode: StringP(conn.cluster.Spec.Config.Cloud.InstanceImage),
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
		SupplementalCreateObjectOptions: &datatypes.Virtual_Guest_SupplementalCreateObjectOptions{
			ImmediateApprovalOnlyFlag: TrueP(),
			PostInstallScriptUri:      StringP("https://raw.githubusercontent.com/pharmer/pharmer/master/cloud/providers/softlayer/startupscript.sh"),
		},
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

	node := api.NodeInfo{
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
		return errors.WithStack(err)
	} else if !success {
		return errors.New("Error deleting virtual guest")
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
		return 0, errors.Errorf("unexpected providerID format: %s, format should be: digitalocean://12345", providerID)
	}

	// since split[0] is actually "digitalocean:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return 0, errors.Errorf("provider name from providerID should be digitalocean: %s", providerID)
	}

	return strconv.Atoi(split[2])
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) reboot(id int) (bool, error) {
	service := conn.virtualServiceClient.Id(id)
	return service.RebootDefault()
}

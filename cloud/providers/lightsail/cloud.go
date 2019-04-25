package lightsail

import (
	"context"
	"fmt"
	"strings"

	. "github.com/appscode/go/context"
	. "github.com/appscode/go/types"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster

	client *lightsail.Lightsail
}

var _ InstanceManager = &cloudConnector{}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	config := &_aws.Config{
		Region:      &cluster.Spec.Cloud.Region,
		Credentials: credentials.NewStaticCredentials(typed.AccessKeyID(), typed.SecretAccessKey(), ""),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	conn := cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  lightsail.New(sess),
	}
	//if ok, msg := conn.IsUnauthorized(); !ok {
	//	return nil, errors.Errorf("credential %s does not have necessary authorization. Reason: %s", cluster.Spec.CredentialName, msg)
	//}
	return &conn, nil
}

func (conn *cloudConnector) DetectInstanceImage() (string, error) {
	b1, err := conn.client.GetBlueprints(&lightsail.GetBlueprintsInput{})
	if err != nil {
		return "", err
	}
	for _, bp := range b1.Blueprints {
		if *bp.Platform == lightsail.InstancePlatformLinuxUnix &&
			*bp.Type == lightsail.BlueprintTypeOs &&
			*bp.Group == "ubuntu" && *bp.Version == "16.04 LTS" {
			return *bp.BlueprintId, nil
		}
	}
	return "", errors.New("can't find `Ubuntu 16.04 LTS` image")
}

func (conn *cloudConnector) WaitForOperation(id string) error {
	attempt := 0
	params := &lightsail.GetOperationInput{
		OperationId: StringP(id),
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		resp, err := conn.client.GetOperation(params)
		if err != nil {
			return false, nil
		}
		status := *resp.Operation.Status
		Logger(conn.ctx).Infof("Attempt %v: operation `%s:%s` is in status `%s`", attempt, *resp.Operation.OperationType, id, status)
		return status == lightsail.OperationStatusCompleted || status == "Succeeded", nil
	})
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, error) {
	_, err := conn.client.GetKeyPair(&lightsail.GetKeyPairInput{
		KeyPairName: StringP(conn.cluster.Spec.Cloud.SSHKeyName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == lightsail.ErrCodeNotFoundException {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Infof("Adding SSH public key")

	_, err := conn.client.ImportKeyPair(&lightsail.ImportKeyPairInput{
		KeyPairName:     StringP(conn.cluster.Spec.Cloud.SSHKeyName),
		PublicKeyBase64: StringP(string(SSHKey(conn.ctx).PublicKey)),
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Info("SSH public key added")
	return nil
}

func (conn *cloudConnector) deleteSSHKey(name string) error {
	_, err := conn.client.DeleteKeyPair(&lightsail.DeleteKeyPairInput{
		KeyPairName: StringP(name),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == lightsail.ErrCodeNotFoundException {
			return nil
		}
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v deleted", conn.cluster.Name)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getReserveIP(ip string) (bool, error) {
	_, err := conn.client.GetStaticIp(&lightsail.GetStaticIpInput{
		StaticIpName: StringP(conn.cluster.Name),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == lightsail.ErrCodeNotFoundException {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) createReserveIP() (string, error) {
	resp, err := conn.client.AllocateStaticIp(&lightsail.AllocateStaticIpInput{
		StaticIpName: StringP(conn.cluster.Name),
	})
	if err != nil {
		return "", err
	}
	err = conn.WaitForOperation(*resp.Operations[0].Id)
	if err != nil {
		return "", err
	}
	sip, err := conn.client.GetStaticIp(&lightsail.GetStaticIpInput{
		StaticIpName: StringP(conn.cluster.Name),
	})
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("New floating ip %v reserved", *sip.StaticIp.IpAddress)
	return *sip.StaticIp.IpAddress, nil
}

func (conn *cloudConnector) assignReservedIP(instanceName string) error {
	_, err := conn.client.AttachStaticIp(&lightsail.AttachStaticIpInput{
		StaticIpName: StringP(conn.cluster.Name),
		InstanceName: StringP(instanceName),
	})
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Reserved ip %v assigned to droplet %v", conn.cluster.Name, instanceName)
	return nil
}

func (conn *cloudConnector) releaseReservedIP() error {
	resp, err := conn.client.ReleaseStaticIp(&lightsail.ReleaseStaticIpInput{
		StaticIpName: StringP(conn.cluster.Name),
	})
	Logger(conn.ctx).Debugln("DO response", resp, " errors", err)
	if err != nil {
		return errors.WithStack(err)
	}
	Logger(conn.ctx).Infof("Floating ip %v deleted", conn.cluster.Name)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	script, err := conn.renderStartupScript(ng, name, token)
	if err != nil {
		return nil, err
	}

	fmt.Println()
	fmt.Println(script)
	fmt.Println()

	params := &lightsail.CreateInstancesInput{
		AvailabilityZone: StringP(conn.cluster.Spec.Cloud.Zone),
		BlueprintId:      StringP(conn.cluster.Spec.Cloud.InstanceImage),
		BundleId:         StringP(ng.Spec.Template.Spec.SKU),
		InstanceNames: []*string{
			StringP(name),
		},
		KeyPairName: StringP(conn.cluster.Spec.Cloud.SSHKeyName),
		UserData:    StringP(script),
	}
	resp, err := conn.client.CreateInstances(params)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Droplet %v created", name)

	if err = conn.WaitForOperation(*resp.Operations[0].Id); err != nil {
		return nil, err
	}

	conn.client.OpenInstancePublicPorts(&lightsail.OpenInstancePublicPortsInput{
		InstanceName: StringP(name),
		PortInfo: &lightsail.PortInfo{
			FromPort: Int64P(6443),
			ToPort:   Int64P(6443),
			Protocol: StringP(lightsail.NetworkProtocolTcp),
		},
	})

	// load again to get IP address assigned
	host, err := conn.client.GetInstance(&lightsail.GetInstanceInput{
		InstanceName: StringP(name),
	})
	if err != nil {
		return nil, err
	}

	return &api.NodeInfo{
		Name:       name,
		ExternalID: *host.Instance.Arn,
		PublicIP:   *host.Instance.PublicIpAddress,
		PrivateIP:  *host.Instance.PrivateIpAddress,
	}, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	instanceId, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return err
	}

	instance, err := conn.instanceByID(instanceId)
	if err != nil {
		return err
	}

	_, err = conn.client.DeleteInstance(&lightsail.DeleteInstanceInput{
		InstanceName: instance.Name,
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Droplet %v deleted", String(instance.Name))
	return nil
}

func (conn *cloudConnector) instanceByID(instanceID string) (*lightsail.Instance, error) {
	host, err := conn.client.GetInstance(&lightsail.GetInstanceInput{
		InstanceName: StringP(instanceID),
	})
	if err != nil {
		return nil, err
	}

	if host.Instance != nil {
		return host.Instance, nil
	}

	return nil, errors.Errorf("Instance with %v not found", instanceID)

}

// dropletIDFromProviderID returns a droplet's ID from providerID.
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: lightsail://droplet-id
func instanceIDFromProviderID(providerID string) (string, error) {
	if providerID == "" {
		return "", errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")

	if len(split) != 3 {
		return "", errors.Errorf("unexpected providerID format: %s, format should be: lightsail://12345", providerID)
	}

	// since split[0] is actually "lightsail:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return "", errors.Errorf("provider name from providerID should be lightsail: %s", providerID)
	}

	return split[2], nil
}

// ---------------------------------------------------------------------------------------------------------------------

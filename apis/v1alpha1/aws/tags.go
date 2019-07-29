package aws

import (
	"fmt"
	"path"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"
)

// Map defines a map of tags.
type Map map[string]string

// Equals returns true if the maps are equal.
func (m Map) Equals(other Map) bool {
	return reflect.DeepEqual(m, other)
}

// HasOwned returns true if the tags contains a tag that marks the resource as owned by the cluster.
func (m Map) HasOwned(cluster string) bool {
	value, ok := m[path.Join(NameKubernetesClusterPrefix, cluster)]
	return ok && ResourceLifecycle(value) == ResourceLifecycleOwned
}

// HasManaged returns true if the map contains NameAWSProviderManaged key set to true.
func (m Map) HasManaged() bool {
	value, ok := m[NameAWSProviderManaged]
	return ok && value == "true"
}

// GetRole returns the Cluster API role for the tagged resource
func (m Map) GetRole() string {
	return m[NameAWSClusterAPIRole]
}

// Difference returns the difference between this map and the other map.
// Items are considered equals if key and value are equals.
func (m Map) Difference(other Map) Map {
	res := make(Map, len(m))

	for key, value := range m {
		if otherValue, ok := other[key]; ok && value == otherValue {
			continue
		}
		res[key] = value
	}

	return res
}

// ResourceLifecycle configures the lifecycle of a resource
type ResourceLifecycle string

const (
	// ResourceLifecycleOwned is the value we use when tagging resources to indicate
	// that the resource is considered owned and managed by the cluster,
	// and in particular that the lifecycle is tied to the lifecycle of the cluster.
	ResourceLifecycleOwned = ResourceLifecycle("owned")

	// ResourceLifecycleShared is the value we use when tagging resources to indicate
	// that the resource is shared between multiple clusters, and should not be destroyed
	// if the cluster is destroyed.
	ResourceLifecycleShared = ResourceLifecycle("shared")

	// NameKubernetesClusterPrefix is the tag name we use to differentiate multiple
	// logically independent clusters running in the same AZ.
	// The tag key = NameKubernetesClusterPrefix + clusterID
	// The tag value is an ownership value
	NameKubernetesClusterPrefix = "kubernetes.io/cluster/"

	// NameAWSProviderPrefix is the tag prefix we use to differentiate
	// cluster-api-provider-aws owned components from other tooling that
	// uses NameKubernetesClusterPrefix
	NameAWSProviderPrefix = "sigs.k8s.io/cluster-api-provider-aws/"

	// NameAWSProviderManaged is the tag name we use to differentiate
	// cluster-api-provider-aws owned components from other tooling that
	// uses NameKubernetesClusterPrefix
	NameAWSProviderManaged = NameAWSProviderPrefix + "managed"

	// NameAWSClusterAPIRole is the tag name we use to mark roles for resources
	// dedicated to this cluster api provider implementation.
	NameAWSClusterAPIRole = NameAWSProviderPrefix + "role"

	// ValueAPIServerRole describes the value for the apiserver role
	ValueAPIServerRole = "apiserver"

	// ValueBastionRole describes the value for the bastion role
	ValueBastionRole = "bastion"

	// ValueCommonRole describes the value for the common role
	ValueCommonRole = "common"

	// ValuePublicRole describes the value for the public role
	ValuePublicRole = "public"

	// ValuePrivateRole describes the value for the private role
	ValuePrivateRole = "private"
)

// ApplyParams are function parameters used to apply tags on an aws resource.
type ApplyParams struct {
	BuildParams
	EC2Client ec2iface.EC2API
}

// Apply tags a resource with tags including the cluster tag.
func Apply(params *ApplyParams) error {
	tags := Build(params.BuildParams)

	awsTags := make([]*ec2.Tag, 0, len(tags))
	for k, v := range tags {
		tag := &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}
		awsTags = append(awsTags, tag)
	}

	createTagsInput := &ec2.CreateTagsInput{
		Resources: aws.StringSlice([]string{params.ResourceID}),
		Tags:      awsTags,
	}

	_, err := params.EC2Client.CreateTags(createTagsInput)
	return errors.Wrapf(err, "failed to tag resource %q in cluster %q", params.ResourceID, params.ClusterName)
}

// Ensure applies the tags if the current tags differ from the params.
func Ensure(current Map, params *ApplyParams) error {
	want := Build(params.BuildParams)
	if !current.Equals(want) {
		return Apply(params)
	}
	return nil
}

// BuildParams is used to build tags around an aws resource.
type BuildParams struct {
	// Lifecycle determines the resource lifecycle.
	Lifecycle ResourceLifecycle

	// ClusterName is the cluster associated with the resource.
	ClusterName string

	// ResourceID is the unique identifier of the resource to be tagged.
	ResourceID string

	// Name is the name of the resource, it's applied as the tag "Name" on AWS.
	// +optional
	Name *string

	// Role is the role associated to the resource.
	// +optional
	Role *string

	// Any additional tags to be added to the resource.
	// +optional
	Additional Map
}

// Build builds tags including the cluster tag and returns them in map form.
func Build(params BuildParams) Map {
	tags := make(Map)
	for k, v := range params.Additional {
		tags[k] = v
	}

	tags[ClusterKey(params.ClusterName)] = string(params.Lifecycle)
	if params.Lifecycle == ResourceLifecycleOwned {
		tags[NameAWSProviderManaged] = "true"
	}

	if params.Role != nil {
		tags[NameAWSClusterAPIRole] = *params.Role
	}

	if params.Name != nil {
		tags["Name"] = *params.Name
	}

	return tags
}

// ClusterKey generates the key for resources associated with a cluster.
func ClusterKey(name string) string {
	return fmt.Sprintf("%s%s", NameKubernetesClusterPrefix, name)
}

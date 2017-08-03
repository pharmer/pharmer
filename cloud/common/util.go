package common

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/system"
	semver "github.com/hashicorp/go-version"
)

var InstanceNotFound = errors.New("Instance not found")
var UnsupportedOperation = errors.New("Unsupported operation")

func SetApps(ctx *contexts.ClusterContext) {
	ctx.Apps = make(map[string]*system.Application)
	ctx.Apps[system.AppKubeSaltbase] = system.NewAppKubernetesSalt(ctx.Provider, ctx.Region, ctx.SaltbaseVersion)
	ctx.Apps[system.AppKubeServer] = system.NewAppKubernetesServer(ctx.Provider, ctx.Region, ctx.KubeServerVersion)
	ctx.Apps[system.AppKubeStarter] = system.NewAppStartKubernetes(ctx.Provider, ctx.Region, ctx.KubeStarterVersion)
	ctx.Apps[system.AppHostfacts] = system.NewAppHostfacts(ctx.Provider, ctx.Region, ctx.HostfactsVersion)
}

func BuildRuntimeConfig(ctx *contexts.ClusterContext) {
	if ctx.EnableThirdPartyResource {
		if ctx.RuntimeConfig == "" {
			ctx.RuntimeConfig = "extensions/v1beta1=true,extensions/v1beta1/thirdpartyresources=true"
		} else {
			ctx.RuntimeConfig += ",extensions/v1beta1=true,extensions/v1beta1/thirdpartyresources=true"
		}
	}

	version, err := semver.NewVersion(ctx.KubeServerVersion)
	if err != nil {
		version, err = semver.NewVersion(ctx.KubeVersion)
		if err != nil {
			return
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		// Enable ScheduledJobs: http://kubernetes.io/docs/user-guide/scheduled-jobs/#prerequisites
		if ctx.EnableScheduledJobResource {
			if ctx.RuntimeConfig == "" {
				ctx.RuntimeConfig = "batch/v2alpha1"
			} else {
				ctx.RuntimeConfig += ",batch/v2alpha1"
			}
		}

		// http://kubernetes.io/docs/admin/authentication/
		if ctx.EnableWebhookTokenAuthentication {
			if ctx.RuntimeConfig == "" {
				ctx.RuntimeConfig = "authentication.k8s.io/v1beta1=true"
			} else {
				ctx.RuntimeConfig += ",authentication.k8s.io/v1beta1=true"
			}
		}

		// http://kubernetes.io/docs/admin/authorization/
		if ctx.EnableWebhookTokenAuthorization {
			if ctx.RuntimeConfig == "" {
				ctx.RuntimeConfig = "authorization.k8s.io/v1beta1=true"
			} else {
				ctx.RuntimeConfig += ",authorization.k8s.io/v1beta1=true"
			}
		}
		if ctx.EnableRBACAuthorization {
			if ctx.RuntimeConfig == "" {
				ctx.RuntimeConfig = "rbac.authorization.k8s.io/v1alpha1=true"
			} else {
				ctx.RuntimeConfig += ",rbac.authorization.k8s.io/v1alpha1=true"
			}
		}
	}
}

func UpgradeRequired(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) bool {
	return ctx.KubeServerVersion != req.KubeletVersion || ctx.SaltbaseVersion != req.SaltbaseVersion || ctx.KubeStarterVersion != req.KubeStarterVersion
}

/*
func NewInstances(ctx *contexts.ClusterContext) (*contexts.ClusterInstances, error) {
	kp := extpoints.KubeProviders.Lookup(ctx.Provider)
	if kp == nil {
		return nil, errors.New().WithMessagef("Missing cloud provider %v", ctx.Provider).Err()
	}
	return contexts.NewInstances(ctx, kp.MatchInstance), nil
}
*/

func SyncAddedInstances(ctx *contexts.ClusterContext, instances []*contexts.KubernetesInstance, purchasePHIDs []string) (int, error) {
	return 0, nil
	/*
		m := make(map[string]*contexts.KubernetesInstance)
		for _, i := range ctx.Instances {
			m[i.ExternalID] = i
		}

		pi := 0
		newAdd := 0
		for _, i := range instances {
			if _, found := m[i.ExternalID]; !found {
				// add to KubernetesInstance table
				i.Role = system.RoleKubernetesPool
				si := &storage.KubernetesInstance{
					KubernetesPHID: ctx.PHID,
					ExternalID:     i.ExternalID,
					ExternalStatus: i.ExternalStatus,
					Name:           i.Name,
					ExternalIP:     i.ExternalIP,
					InternalIP:     i.InternalIP,
					SKU:            i.SKU,
					Role:           i.Role,
					Status:         i.Status,
				}
				if has, _ := ctx.Store.Engine.Get(si); has {
					billing.NewController(ctx.Store).FailPurchase(purchasePHIDs[pi])
					pi++
					continue
				}

				si.PHID = i.PHID
				if _, err := ctx.Store.Engine.Insert(si); err != nil {
					return pi, errors.FromErr(err).WithContext(ctx).Err()
				}
				ctx.Instances = append(ctx.Instances, i)
				// add billing
				if err := AddBillingForNode(ctx, i, purchasePHIDs[pi]); err != nil {
					return pi, errors.FromErr(err).WithContext(ctx).Err()
				}
				pi++
				newAdd++
			}
		}
		ctx.NumNodes += int64(newAdd)

		return pi, ctx.UpdateNodeCount()
	*/
}

func SyncDeletedInstances(ctx *contexts.ClusterContext, sku string, instances []*contexts.KubernetesInstance) error {
	return nil
	/*
		m := make(map[string]*contexts.KubernetesInstance)
		// Validate all instances are from same sku
		for _, i := range instances {
			if i.SKU != sku {
				return errors.New(fmt.Sprintf("Cluster %v's instance %v has sku %v but expected %v", ctx.Name, i.ExternalID, i.SKU, sku)).WithContext(ctx).InvalidData()
			}
			m[i.ExternalID] = i
		}

		ctrl := billing.NewController(ctx.Store)

		// https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
		fi := ctx.Instances[:0]
		deletedNode := 0
		for _, i := range ctx.Instances {
			if _, found := m[i.ExternalID]; !found && i.SKU == sku && i.Role == system.RoleKubernetesPool {
				updates := &storage.KubernetesInstance{Status: storage.KubernetesInstanceStatus_Deleted}
				cond := &storage.KubernetesInstance{PHID: i.PHID}
				if _, err := ctx.Store.Engine.Update(updates, cond); err != nil {
					return errors.FromErr(err).WithContext(ctx).Err()
				}
				i.Status = storage.KubernetesInstanceStatus_Deleted
				ki := &storage.Purchase{
					ObjectPHID: i.PHID,
					Status:     storage.ChargeStatus_Close,
				}
				if has, _ := ctx.Store.Engine.Get(ki); has {
					continue
				}

				if err := ctrl.ClosePurchase(i.PHID); err != nil {
					return errors.FromErr(err).WithContext(ctx).Err()
				}
				deletedNode++
			} else {
				fi = append(fi, i)
			}
		}
		ctx.Instances = fi
		ctx.NumNodes -= int64(deletedNode)
		return ctx.UpdateNodeCount()
	*/
}

func SyncClusterContextWithNumNode(ctx *contexts.ClusterContext, nodeAtdb, nodeInc int64, sku string) error {
	return nil
	/*
		kv, err := ctx.Store.GetKubernetesContext(ctx.ContextVersion)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		decodeContext, err := storage.NewSecEnvelope(kv.Context)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		dc, err := decodeContext.ValBytes()
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}

		dbContext := ctx

		err = json.Unmarshal(dc, dbContext)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		nodeNum := nodeAtdb + nodeInc
		if dbContext.NumNodes != nodeNum {
			ctx.NumNodes = nodeNum

		}
		return ctx.UpdateNodeCount()
	*/
}

package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appscode/go/log"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
)

func GetPreconditionFunc() []mergepatch.PreconditionFunc {
	preconditions := []mergepatch.PreconditionFunc{
		mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"),
		mergepatch.RequireMetadataKeyUnchanged("name"),
		mergepatch.RequireMetadataKeyUnchanged("namespace"),
		mergepatch.RequireKeyUnchanged("status"),
	}
	return preconditions
}

//TODO: Add restricted field
var PreconditionSpecField = map[string][]string{
	api.ResourceKindCluster: {
		"metadata",
		"spec.cloud",
		"spec.api",
		"spec.masterInternalIp",
		"spec.masterDiskID",
		"spec.networking.dnsDomain",
	},
	api.ResourceKindNodeGroup: {
		"metadata",
		"template.spec.externalIPType",
	},
	cloudapi.ResourceKindCredential: {
		"metadata",
	},
}

func GetConditionalPreconditionFunc(kind string) []mergepatch.PreconditionFunc {
	preconditions := []mergepatch.PreconditionFunc{}

	if fields, found := PreconditionSpecField[kind]; found {
		for _, field := range fields {
			preconditions = append(preconditions,
				RequireChainKeyUnchanged(field),
			)
		}
	}

	return preconditions
}

func checkChainKeyUnchanged(key string, mapData map[string]interface{}) bool {
	keys := strings.Split(key, ".")
	val, ok := mapData[keys[0]]
	if !ok || len(keys) == 1 {
		return !ok
	}

	newKey := strings.Join(keys[1:], ".")
	return checkChainKeyUnchanged(newKey, val.(map[string]interface{}))
}

func RequireChainKeyUnchanged(key string) mergepatch.PreconditionFunc {
	return func(patch interface{}) bool {
		patchMap, ok := patch.(map[string]interface{})
		if !ok {
			log.Infoln("Invalid data")
			return true
		}
		return checkChainKeyUnchanged(key, patchMap)
	}
}

func CheckConditionalPrecondition(patchData []byte, fns ...mergepatch.PreconditionFunc) error {
	patch := make(map[string]interface{})
	if err := json.Unmarshal(patchData, &patch); err != nil {
		return err
	}
	for _, fn := range fns {
		if !fn(patch) {
			return newErrPreconditionFailed(patch)
		}
	}
	return nil
}

func newErrPreconditionFailed(target map[string]interface{}) errPreconditionFailed {
	s := fmt.Sprintf("precondition failed for: %v", target)
	return errPreconditionFailed{s}
}

type errPreconditionFailed struct {
	message string
}

func (err errPreconditionFailed) Error() string {
	return err.message
}

func IsPreconditionFailed(err error) bool {
	_, ok := err.(errPreconditionFailed)
	return ok
}

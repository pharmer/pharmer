/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/appscode/go/log"
	"k8s.io/apimachinery/pkg/util/mergepatch"
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

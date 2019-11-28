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
package describer

import (
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/describe"
)

type Describer interface {
	Describe(object runtime.Object, describerSettings describe.DescriberSettings) (output string, err error)
}

func NewDescriber() Describer {
	return newHumanReadableDescriber()
}

type handlerEntry struct {
	describeFunc reflect.Value
}

type humanReadableDescriber struct {
	handlerMap map[reflect.Type]*handlerEntry
}

func newHumanReadableDescriber() *humanReadableDescriber {
	describer := &humanReadableDescriber{
		handlerMap: make(map[reflect.Type]*handlerEntry),
	}
	describer.addDefaultHandlers()
	return describer
}

func (h *humanReadableDescriber) addDefaultHandlers() {
	_ = h.Handler(h.describeCluster)
}

func (h *humanReadableDescriber) Handler(describeFunc interface{}) error {
	describeFuncValue := reflect.ValueOf(describeFunc)
	if err := h.validateDescribeHandlerFunc(describeFuncValue); err != nil {
		klog.Errorf("Unable to add describe handler: %v", err)
		return err
	}

	objType := describeFuncValue.Type().In(0)

	h.handlerMap[objType] = &handlerEntry{
		describeFunc: describeFuncValue,
	}
	return nil
}

func (h *humanReadableDescriber) validateDescribeHandlerFunc(describeFunc reflect.Value) error {
	if describeFunc.Kind() != reflect.Func {
		return errors.Errorf("invalid describe handler. %#v is not a function", describeFunc)
	}
	funcType := describeFunc.Type()
	if funcType.NumIn() != 2 || funcType.NumOut() != 2 {
		return errors.Errorf("invalid describe handler." +
			"Must accept 2 parameters and return 2 value.")
	}

	if funcType.In(1) != reflect.TypeOf((describe.DescriberSettings)(describe.DescriberSettings{})) ||
		funcType.Out(0) != reflect.TypeOf((string)("")) ||
		funcType.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return errors.Errorf("invalid describe handler. The expected signature is: "+
			"func handler(item %v, describerSettings *printers.DescriberSettings) (string, error)", funcType.In(0))
	}
	return nil
}

func (h *humanReadableDescriber) Describe(obj runtime.Object, describerSettings describe.DescriberSettings) (string, error) {
	t := reflect.TypeOf(obj)
	if handler := h.handlerMap[t]; handler != nil {
		args := []reflect.Value{reflect.ValueOf(obj), reflect.ValueOf(describerSettings)}
		resultValue := handler.describeFunc.Call(args)
		if err := resultValue[1].Interface(); err != nil {
			return resultValue[0].Interface().(string), err.(error)
		}

		return resultValue[0].Interface().(string), nil
	}

	return "", errors.Errorf(`pharmer doesn't support: "%v"`, t)
}

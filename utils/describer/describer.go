package describer

import (
	"context"
	"reflect"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/printers"
)

type Describer interface {
	Describe(object runtime.Object, describerSettings *printers.DescriberSettings) (output string, err error)
}

func NewDescriber(ctx context.Context, owner string) Describer {
	return newHumanReadableDescriber(ctx, owner)
}

type handlerEntry struct {
	describeFunc reflect.Value
	args         []reflect.Value
}

type humanReadableDescriber struct {
	handlerMap map[reflect.Type]*handlerEntry
	ctx        context.Context
	owner      string
}

func newHumanReadableDescriber(ctx context.Context, owner string) *humanReadableDescriber {
	describer := &humanReadableDescriber{
		handlerMap: make(map[reflect.Type]*handlerEntry),
		ctx:        ctx,
		owner:      owner,
	}
	describer.addDefaultHandlers()
	return describer
}

func (h *humanReadableDescriber) addDefaultHandlers() {
	h.Handler(h.describeCluster)
}

func (h *humanReadableDescriber) Handler(describeFunc interface{}) error {
	describeFuncValue := reflect.ValueOf(describeFunc)
	if err := h.validateDescribeHandlerFunc(describeFuncValue); err != nil {
		glog.Errorf("Unable to add describe handler: %v", err)
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

	if funcType.In(1) != reflect.TypeOf((*printers.DescriberSettings)(nil)) ||
		funcType.Out(0) != reflect.TypeOf((string)("")) ||
		funcType.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return errors.Errorf("invalid describe handler. The expected signature is: "+
			"func handler(item %v, describerSettings *printers.DescriberSettings) (string, error)", funcType.In(0))
	}
	return nil
}

func (h *humanReadableDescriber) Describe(obj runtime.Object, describerSettings *printers.DescriberSettings) (string, error) {
	t := reflect.TypeOf(obj)
	if handler := h.handlerMap[t]; handler != nil {
		args := []reflect.Value{reflect.ValueOf(obj), reflect.ValueOf(describerSettings)}
		resultValue := handler.describeFunc.Call(args)
		if err := resultValue[1].Interface(); err != nil {
			return resultValue[0].Interface().(string), err.(error)
		}

		return resultValue[0].Interface().(string), nil
	}

	return "", errors.Errorf(`kubedb doesn't support: "%v"`, t)
}

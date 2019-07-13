package printer

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// ref: k8s.io/kubernetes/pkg/kubectl/resource_printer.go

const (
	tabwriterMinWidth = 10
	tabwriterWidth    = 4
	tabwriterPadding  = 3
	tabwriterPadChar  = ' '
	tabwriterFlags    = 0
)

type handlerEntry struct {
	printFunc reflect.Value
}

type PrintOptions struct {
	Wide bool
}

type HumanReadablePrinter struct {
	handlerMap        map[reflect.Type]*handlerEntry
	options           PrintOptions
	lastType          reflect.Type
	enablePrintHeader bool
}

func NewHumanReadablePrinter(options PrintOptions) *HumanReadablePrinter {
	printer := &HumanReadablePrinter{
		handlerMap:        make(map[reflect.Type]*handlerEntry),
		options:           options,
		enablePrintHeader: true,
	}
	printer.addDefaultHandlers()
	return printer
}

func ShortHumanDuration(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds <= 0 {
		return fmt.Sprintf("0s")
	} else if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if minutes := int(d.Minutes()); minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	} else if hours := int(d.Hours()); hours < 24 {
		return fmt.Sprintf("%dh", hours)
	} else if hours < 24*364 {
		return fmt.Sprintf("%dd", hours/24)
	}
	return fmt.Sprintf("%dy", int(d.Hours()/24/365))
}

func (h *HumanReadablePrinter) addDefaultHandlers() {
	_ = h.Handler(h.printCluster)
	_ = h.Handler(h.printCredential)
	_ = h.Handler(h.printMachineSet)
}

func (h *HumanReadablePrinter) PrintHeader(enable bool) {
	h.enablePrintHeader = enable
}

func (h *HumanReadablePrinter) Handler(printFunc interface{}) error {
	printFuncValue := reflect.ValueOf(printFunc)
	if err := h.validatePrintHandlerFunc(printFuncValue); err != nil {
		klog.Errorf("Unable to add print handler: %v", err)
		return err
	}

	objType := printFuncValue.Type().In(0)

	h.handlerMap[objType] = &handlerEntry{
		printFunc: printFuncValue,
	}
	return nil
}

func (h *HumanReadablePrinter) validatePrintHandlerFunc(printFunc reflect.Value) error {
	if printFunc.Kind() != reflect.Func {
		return errors.Errorf("invalid print handler. %#v is not a function", printFunc)
	}
	funcType := printFunc.Type()
	if funcType.NumIn() != 3 || funcType.NumOut() != 1 {
		return errors.Errorf("invalid print handler." +
			"Must accept 3 parameters and return 1 value.")
	}
	if funcType.In(1) != reflect.TypeOf((*io.Writer)(nil)).Elem() ||
		funcType.In(2) != reflect.TypeOf((*PrintOptions)(nil)).Elem() ||
		funcType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return errors.Errorf("invalid print handler. The expected signature is: "+
			"func handler(obj %v, w io.Writer, options PrintOptions) error", funcType.In(0))
	}
	return nil
}

func getColumns(t reflect.Type) []string {
	columns := make([]string, 0)
	columns = append(columns, "NAME")
	switch t.String() {
	case "*v1alpha1.Cluster":
		columns = append(columns, "PROVIDER")
		columns = append(columns, "ZONE")
		columns = append(columns, "VERSION")
		columns = append(columns, "RUNNING SINCE")
		columns = append(columns, "STATUS")
	case "*v1alpha1.NodeGroup":
		columns = append(columns, "Cluster")
		columns = append(columns, "Node")
		columns = append(columns, "SKU")
	case "*v1alpha1.Credential":
		columns = append(columns, "Provider")
		columns = append(columns, "Data")
	case "*cluster.k8s.io/v1alpha1":
		columns = append(columns, "Cluster")
		columns = append(columns, "Node")
		columns = append(columns, "SKU")
	}
	return columns
}

func (h *HumanReadablePrinter) printCluster(item *api.Cluster, w io.Writer, options PrintOptions) (err error) {
	name := item.Name

	if _, err = fmt.Fprintf(w, "%s\t", name); err != nil {
		return
	}

	if _, err = fmt.Fprintf(w, "%s\t", item.Spec.Config.Cloud.CloudProvider); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "%s\t", item.Spec.Config.Cloud.Zone); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "%s\t", item.Spec.Config.KubernetesVersion); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "%s\t", TranslateTimestamp(item.CreationTimestamp)); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "%s\t", item.Status.Phase); err != nil {
		return
	}
	return PrintNewline(w)
}

func (h *HumanReadablePrinter) printMachineSet(item *clusterv1.MachineSet, w io.Writer, options PrintOptions) error {
	name := item.Name

	if _, err := fmt.Fprintf(w, "%s\t", name); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\t", item.ClusterName); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%v\t", item.Spec.Template.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\t", item.Spec.Template.Spec.Namespace); err != nil {
		return err
	}
	return PrintNewline(w)
}

func (h *HumanReadablePrinter) printCredential(item *cloudapi.Credential, w io.Writer, options PrintOptions) error {
	name := item.Name

	if _, err := fmt.Fprintf(w, "%s\t", name); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\t", item.Spec.Provider); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\t", credential.CommonSpec(item.Spec).String()); err != nil {
		return err
	}
	return PrintNewline(w)
}

func (h *HumanReadablePrinter) PrintObj(obj runtime.Object, output io.Writer) error {
	w, found := output.(*tabwriter.Writer)
	if !found {
		w = GetNewTabWriter(output)
		defer w.Flush()
	}

	t := reflect.TypeOf(obj)
	if handler := h.handlerMap[t]; handler != nil {

		if t != h.lastType || h.enablePrintHeader {
			headers := getColumns(t)
			if h.lastType != nil {
				if err := PrintNewline(w); err != nil {
					return err
				}
			}
			if err := h.printHeader(headers, w); err != nil {
				return err
			}
			h.lastType = t
			h.enablePrintHeader = false
		}
		args := []reflect.Value{reflect.ValueOf(obj), reflect.ValueOf(w), reflect.ValueOf(h.options)}
		resultValue := handler.printFunc.Call(args)[0]
		if resultValue.IsNil() {
			return nil
		}
		return resultValue.Interface().(error)
	}

	return errors.Errorf(`pharmer doesn't support: "%v"`, t)
}

func (h *HumanReadablePrinter) HandledResources() []string {
	return []string{}
}

func (h *HumanReadablePrinter) AfterPrint(io.Writer, string) error {
	return nil
}

func (h *HumanReadablePrinter) IsGeneric() bool {
	return false
}

func (h *HumanReadablePrinter) printHeader(columnNames []string, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s\n", strings.Join(columnNames, "\t")); err != nil {
		return err
	}
	return nil
}

func PrintNewline(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}
	return nil
}

func TranslateTimestamp(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	return ShortHumanDuration(time.Since(timestamp.Time))
}

func GetNewTabWriter(output io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(
		output,
		tabwriterMinWidth,
		tabwriterWidth,
		tabwriterPadding,
		tabwriterPadChar,
		tabwriterFlags,
	)
}

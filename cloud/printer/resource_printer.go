package printer

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/appscode/pharmer/api"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

// ref: k8s.io/kubernetes/pkg/kubectl/resource_printer.go

const (
	tabwriterMinWidth = 10
	tabwriterWidth    = 4
	tabwriterPadding  = 3
	tabwriterPadChar  = ' '
	tabwriterFlags    = 0
	statusUnknown     = "Unknown"
)

type handlerEntry struct {
	printFunc reflect.Value
	args      []reflect.Value
}

type PrintOptions struct {
	WithNamespace bool
	WithKind      bool
	Wide          bool
	ShowAll       bool
	ShowLabels    bool
	Kind          string
}

type HumanReadablePrinter struct {
	handlerMap   map[reflect.Type]*handlerEntry
	options      PrintOptions
	lastType     reflect.Type
	hiddenObjNum int
}

func NewHumanReadablePrinter(options PrintOptions) *HumanReadablePrinter {
	printer := &HumanReadablePrinter{
		handlerMap: make(map[reflect.Type]*handlerEntry),
		options:    options,
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
	h.Handler(h.printElastic)
}

func (h *HumanReadablePrinter) Handler(printFunc interface{}) error {
	printFuncValue := reflect.ValueOf(printFunc)
	if err := h.validatePrintHandlerFunc(printFuncValue); err != nil {
		glog.Errorf("Unable to add print handler: %v", err)
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
		return fmt.Errorf("invalid print handler. %#v is not a function", printFunc)
	}
	funcType := printFunc.Type()
	if funcType.NumIn() != 3 || funcType.NumOut() != 1 {
		return fmt.Errorf("invalid print handler." +
			"Must accept 3 parameters and return 1 value.")
	}
	if funcType.In(1) != reflect.TypeOf((*io.Writer)(nil)).Elem() ||
		funcType.In(2) != reflect.TypeOf((*PrintOptions)(nil)).Elem() ||
		funcType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("invalid print handler. The expected signature is: "+
			"func handler(obj %v, w io.Writer, options PrintOptions) error", funcType.In(0))
	}
	return nil
}

func getColumns(options PrintOptions, t reflect.Type) []string {
	columns := make([]string, 0)
	columns = append(columns, "NAME")
	switch t.String() {
	case "*api.Cluster":
		columns = append(columns, "PROVIDER")
		columns = append(columns, "ZONE")
		columns = append(columns, "VERSION")
		columns = append(columns, "RUNNING SINCE")
	}
	return columns
}

func (h *HumanReadablePrinter) printElastic(item *api.Cluster, w io.Writer, options PrintOptions) (err error) {
	name := formatResourceName(options.Kind, item.Name, options.WithKind)

	if _, err = fmt.Fprintf(w, "%s\t", name); err != nil {
		return
	}

	if _, err = fmt.Fprintf(w, "%s\t", item.Spec.Cloud.CloudProvider); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "%s\t", item.Spec.Cloud.Zone); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "%s\t", item.Spec.KubernetesVersion); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "%s\t", TranslateTimestamp(item.CreationTimestamp)); err != nil {
		return
	}

	return
}

func (h *HumanReadablePrinter) PrintObj(obj runtime.Object, output io.Writer) error {
	w, found := output.(*tabwriter.Writer)
	if !found {
		w = GetNewTabWriter(output)
		defer w.Flush()
	}

	t := reflect.TypeOf(obj)
	if handler := h.handlerMap[t]; handler != nil {
		if t != h.lastType {
			headers := getColumns(h.options, t)
			if h.lastType != nil {
				printNewline(w)
			}
			h.printHeader(headers, w)
			h.lastType = t
		}
		args := []reflect.Value{reflect.ValueOf(obj), reflect.ValueOf(w), reflect.ValueOf(h.options)}
		resultValue := handler.printFunc.Call(args)[0]
		if resultValue.IsNil() {
			return nil
		}
		return resultValue.Interface().(error)
	}

	return fmt.Errorf(`kubedb doesn't support: "%v"`, t)
}

func (h *HumanReadablePrinter) HandledResources() []string {
	return []string{}
}

func (h *HumanReadablePrinter) AfterPrint(io.Writer, string) error {
	return nil
}

func (h *HumanReadablePrinter) IsGeneric() bool {
	return true
}

func (h *HumanReadablePrinter) printHeader(columnNames []string, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s\n", strings.Join(columnNames, "\t")); err != nil {
		return err
	}
	return nil
}

func printNewline(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}
	return nil
}

func formatResourceName(kind, name string, withKind bool) string {
	if !withKind || kind == "" {
		return name
	}

	return kind + "/" + name
}

func (h *HumanReadablePrinter) GetResourceKind() string {
	return h.options.Kind
}

func (h *HumanReadablePrinter) EnsurePrintWithKind(kind string) {
	h.options.WithKind = true
	h.options.Kind = kind
}

func appendAllLabels(showLabels bool, itemLabels map[string]string) string {
	var buffer bytes.Buffer

	if showLabels {
		buffer.WriteString(fmt.Sprint("\t"))
		buffer.WriteString(labels.FormatLabels(itemLabels))
	}
	buffer.WriteString("\n")

	return buffer.String()
}

func TranslateTimestamp(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	return ShortHumanDuration(time.Now().Sub(timestamp.Time))
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
package printer

import (
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/printers"
)

// ref: k8s.io/kubernetes/pkg/kubectl/resource_printer.go

func NewPrinter(format string) (printers.ResourcePrinter, error) {
	humanReadablePrinter := NewHumanReadablePrinter(PrintOptions{
		Wide: format == "wide",
	})

	switch format {
	case "json":
		return &printers.JSONPrinter{}, nil
	case "yaml":
		return &printers.YAMLPrinter{}, nil
	case "wide":
		fallthrough
	case "":
		return humanReadablePrinter, nil
	default:
		return nil, errors.Errorf("output format %q not recognized", format)
	}
}

type editPrinterOptions struct {
	Printer   printers.ResourcePrinter
	Ext       string
	AddHeader bool
}

func NewEditPrinter(format string) (*editPrinterOptions, error) {
	switch format {
	case "json":
		return &editPrinterOptions{
			Printer:   &printers.JSONPrinter{},
			Ext:       ".json",
			AddHeader: true,
		}, nil
		// If flag -o is not specified, use yaml as default
	case "yaml", "":
		return &editPrinterOptions{
			Printer:   &printers.YAMLPrinter{},
			Ext:       ".yaml",
			AddHeader: true,
		}, nil
	default:
		return nil, errors.New("The flag 'output' must be one of yaml|json")
	}
}

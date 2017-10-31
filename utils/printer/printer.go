package printer

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/printers"
)

// ref: k8s.io/kubernetes/pkg/kubectl/resource_printer.go

func NewPrinter(cmd *cobra.Command) (printers.ResourcePrinter, error) {
	f := cmd.Flags().Lookup("output")
	humanReadablePrinter := NewHumanReadablePrinter(PrintOptions{
		Wide: f != nil && f.Value != nil && f.Value.String() == "wide",
	})

	format, _ := cmd.Flags().GetString("output")

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
		return nil, fmt.Errorf("output format %q not recognized", format)
	}
}

type editPrinterOptions struct {
	Printer   printers.ResourcePrinter
	Ext       string
	AddHeader bool
}

func NewEditPrinter(cmd *cobra.Command) (*editPrinterOptions, error) {
	switch format, _ := cmd.Flags().GetString("output"); format {
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

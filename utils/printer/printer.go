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
package printer

import (
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/printers"
)

// ref: k8s.io/kubectl/resource_printer.go

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

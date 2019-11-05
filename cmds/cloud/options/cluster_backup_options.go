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
package options

import (
	"strings"

	"github.com/appscode/go/flags"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterBackupConfig struct {
	ClusterName string
	BackupDir   string
	Sanitize    bool
}

func NewClusterBackupConfig() *ClusterBackupConfig {
	return &ClusterBackupConfig{
		ClusterName: "",
		Sanitize:    false,
		BackupDir:   "",
	}
}

func (c *ClusterBackupConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Sanitize, "sanitize", c.Sanitize, " Sanitize fields in YAML")
	fs.StringVar(&c.BackupDir, "backup-dir", c.BackupDir, "Directory where yaml files will be saved")

}

func (c *ClusterBackupConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	flags.EnsureRequiredFlags(cmd, "backup-dir")

	if len(args) == 0 {
		return errors.New("missing cluster name")
	}
	if len(args) > 1 {
		return errors.New("multiple cluster name provided")
	}
	c.ClusterName = strings.ToLower(args[0])
	return nil
}

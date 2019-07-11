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

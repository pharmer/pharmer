package options

import "github.com/spf13/pflag"

type ClusterOperation struct {
	OperationId string `json:"operation_id"`
}

func NewClusterOperation() *ClusterOperation {
	return &ClusterOperation{}
}

func (c *ClusterOperation) AddFlags(fs *pflag.FlagSet)  {
	fs.StringVar(&c.OperationId, "operation-id", c.OperationId, "Operation id")
}

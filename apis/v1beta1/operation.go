package v1beta1

type OperationState int

const (
	OperationDone OperationState = iota // 0
	OperationPending
	OperationRunning
)

const (
	JobStatus_Requested = "REQUESTED"
	JobStatus_Running   = "RUNNING"
	JobStatus_Done      = "DONE"
	JobStatus_Failed    = "FAILED"
)

type Operation struct {
	ID        int64  `xorm:"pk autoincr 'id'"`
	UserID    int64  `xorm:"UNIQUE(s) 'user_id'"`
	ClusterID int64  `xorm:"UNIQUE(s) 'cluster_id'"`
	Code      string `xorm:"UNIQUE(s)"`
	State     OperationState
}

func (Operation) TableName() string {
	return "ac_operation"
}

package contexts

type NotificationContext struct {
	Notifier `json:"-"`
}
type NotificationMessage struct {
	Status  string `json:"status,omitempty"`
	Details string `json:"details,omitempty"`
}
type Notifier interface {
	//return fingerPrint and error
	Notify(status string, details string) (string, error)
	StoreAndNotify(status string, details string) (string, error)
}

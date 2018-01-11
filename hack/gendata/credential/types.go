package credential

const (
	GCEServiceAccount    = "serviceAccount"
	GCEProjectID         = "projectID"
)

type CredentialSpec struct {
	Provider string            `json:"provider" protobuf:"bytes,1,opt,name=provider"`
	Data     map[string]string `json:"data" protobuf:"bytes,2,rep,name=data"`
}
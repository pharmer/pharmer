package api

// ConfigMap is a map[string]string that implements
// the Config method.
type ConfigMap map[string]string

// Config gets a string configuration value and a
// bool indicating whether the value was present or not.
func (c ConfigMap) Config(name string) (string, bool) {
	val, ok := c[name]
	return val, ok
}

type Credential struct {
	TypeMeta   `json:",inline,omitempty"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       CredentialSpec `json:"spec,omitempty"`
}

type CredentialSpec struct {
	Provider string    `json:"provider"`
	Data     ConfigMap `json:"config"`
}

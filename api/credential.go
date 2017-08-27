package api

type Credential struct {
	TypeMeta   `json:",inline,omitempty"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       CredentialSpec `json:"spec,omitempty"`
}

type CredentialSpec struct {
	Provider string            `json:"provider"`
	Data     map[string]string `json:"data"`
}

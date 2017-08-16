package credential

type Credential interface {
	IsValid() bool
	AsMap() map[string]string
}

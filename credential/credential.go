package credential

type AWS struct {
	AccessKeyId     string `json:"access_key_id" envconfig:"AWS_ACCESS_KEY_ID" form:"aws_access_key_id"`
	SecretAccessKey string `json:"secret_access_key" envconfig:"AWS_SECRET_ACCESS_KEY" form:"aws_secret_access_key"`
	Region          string `json:"region" envconfig:"AWS_REGION" form:"aws_region"`
}

type Azure struct {
	TenantId       string `json:"tenant_id" envconfig:"AZURE_TENANT_ID" form:"azure_tenant_id"`
	SubscriptionId string `json:"subscription_id" envconfig:"AZURE_SUBSCRIPTION_ID" form:"azure_subscription_id"`
	ClientId       string `json:"client_id" envconfig:"AZURE_CLIENT_ID" form:"azure_client_id"`
	ClientSecret   string `json:"client_secret" envconfig:"AZURE_CLIENT_SECRET" form:"azure_client_secret"`
	ResourceGroup  string `json:"resource_group" envconfig:"AZURE_RESOURCE_GROUP" form:"azure_resource_group"`
}

type DigitalOcean struct {
	AuthToken string `json:"auth_token" envconfig:"DO_AUTH_TOKEN" form:"digitalocean_auth_token"`
}

type Google struct {
	Project        string `json:"project" envconfig:"GCE_PROJECT"  form:"gcloud_project"`
	CredentialFile string `json:"credential_file" ignore:"true" form:"-"`
	CredentialJson string `json:"-" ignore:"true" form:"gcloud_credential_json"`
	JsonKey        []byte `json:"-" ignore:"true"  form:"-"`
}

type Linode struct {
	ApiKey string `json:"api_key" envconfig:"LINODE_API_KEY" form:"linode_api_key"`
}

type Vultr struct {
	ApiKey string `json:"api_key" envconfig:"VULTR_API_KEY" form:"vultr_api_key"`
}

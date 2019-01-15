package cmds

import (
	"io/ioutil"
	"time"

	"github.com/appscode/go/term"
	"github.com/graymeta/stow/azure"
	gcs "github.com/graymeta/stow/google"
	"github.com/graymeta/stow/local"
	"github.com/graymeta/stow/s3"
	"github.com/graymeta/stow/swift"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/credential"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type setContextRequest struct {
	Name                string
	Provider            string
	Prefix              string
	s3ConfigAccessKeyID string //cred
	s3ConfigSecretKey   string //cred

	s3StoreEndpoint string //store
	s3StoreBucket   string //store

	gcsConfigJSONKeyPath string //cred
	gcsConfigProjectId   string //cred

	gcsStoreBucket string //store

	azureConfigAccount string //cred
	azureConfigKey     string //cred

	azureStoreContainer string //store

	localConfigKeyPath       string
	swiftConfigKey           string
	swiftConfigTenantAuthURL string
	swiftConfigTenantName    string
	swiftConfigUsername      string
	swiftConfigDomain        string
	swiftConfigRegion        string
	swiftConfigTenantId      string
	swiftConfigTenantDomain  string
	swiftConfigStorageURL    string
	swiftConfigAuthToken     string

	swiftStoreContainer string //store

	pgDatabaseName string
	pgHost         string
	pgPort         int64
	pgUser         string
	pgPassword     string
}

func newCmdCreate() *cobra.Command {
	req := &setContextRequest{}

	setCmd := &cobra.Command{
		Use:               "set-context",
		Short:             "Create  config object",
		Example:           `pharmer config set-context`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			setContext(req, cfgFile)
		},
	}
	setCmd.Flags().StringVar(&req.Provider, "provider", "", "Cloud storage provider")

	setCmd.Flags().StringVar(&req.s3ConfigAccessKeyID, s3.Kind+"."+s3.ConfigAccessKeyID, "", "S3 config access key id")
	setCmd.Flags().StringVar(&req.s3ConfigSecretKey, s3.Kind+"."+s3.ConfigSecretKey, "", "S3 config secret key")

	setCmd.Flags().StringVar(&req.s3StoreEndpoint, s3.Kind+"."+s3.ConfigEndpoint, "", "S3 storage endpoint")
	setCmd.Flags().StringVar(&req.s3StoreBucket, s3.Kind+".bucket", "", "S3 store bucket")

	setCmd.Flags().StringVar(&req.gcsConfigJSONKeyPath, gcs.Kind+".json_key_path", "", "GCS config json key path")
	setCmd.Flags().StringVar(&req.gcsConfigProjectId, gcs.Kind+"."+gcs.ConfigProjectId, "", "GCS config project id")

	setCmd.Flags().StringVar(&req.gcsStoreBucket, gcs.Kind+".bucket", "", "GCS config scopes")

	setCmd.Flags().StringVar(&req.azureConfigAccount, azure.Kind+"."+azure.ConfigAccount, "", "Azure config account")
	setCmd.Flags().StringVar(&req.azureConfigKey, azure.Kind+"."+azure.ConfigKey, "", "Azure config key")

	setCmd.Flags().StringVar(&req.azureStoreContainer, azure.Kind+".container", "", "Azure container name")

	setCmd.Flags().StringVar(&req.localConfigKeyPath, local.Kind+"."+local.ConfigKeyPath, "", "Local config key path")

	setCmd.Flags().StringVar(&req.swiftConfigKey, swift.Kind+"."+swift.ConfigKey, "", "Swift config key")
	setCmd.Flags().StringVar(&req.swiftConfigTenantAuthURL, swift.Kind+"."+swift.ConfigTenantAuthURL, "", "Swift teanant auth url")
	setCmd.Flags().StringVar(&req.swiftConfigTenantName, swift.Kind+"."+swift.ConfigTenantName, "", "Swift tenant name")
	setCmd.Flags().StringVar(&req.swiftConfigUsername, swift.Kind+"."+swift.ConfigUsername, "", "Swift username")
	setCmd.Flags().StringVar(&req.swiftConfigDomain, swift.Kind+"."+swift.ConfigDomain, "", "Swift domain")
	setCmd.Flags().StringVar(&req.swiftConfigRegion, swift.Kind+"."+swift.ConfigRegion, "", "Swift region")
	setCmd.Flags().StringVar(&req.swiftConfigTenantId, swift.Kind+"."+swift.ConfigTenantId, "", "Swift TenantId")
	setCmd.Flags().StringVar(&req.swiftConfigTenantDomain, swift.Kind+"."+swift.ConfigTenantDomain, "", "Swift TenantDomain")
	setCmd.Flags().StringVar(&req.swiftConfigStorageURL, swift.Kind+"."+swift.ConfigStorageURL, "", "Swift StorageURL")
	setCmd.Flags().StringVar(&req.swiftConfigAuthToken, swift.Kind+"."+swift.ConfigAuthToken, "", "Swift AuthToken")

	setCmd.Flags().StringVar(&req.swiftStoreContainer, swift.Kind+".container", "", "Swift container name")

	setCmd.Flags().StringVar(&req.pgDatabaseName, "pg.db-name", "", "Postgres databases name")
	setCmd.Flags().StringVar(&req.pgHost, "pg.host", "", "Postgres host address")
	setCmd.Flags().Int64Var(&req.pgPort, "pg.port", int64(5432), "Postgres port number")
	setCmd.Flags().StringVar(&req.pgUser, "pg.user", "", "Postgres database user")
	setCmd.Flags().StringVar(&req.pgPassword, "pg.password", "", "Postgres user password")

	return setCmd
}

func setContext(req *setContextRequest, configPath string) {
	pc := &api.PharmerConfig{
		Context: "default",
		TypeMeta: metav1.TypeMeta{
			Kind: "PharmerConfig",
		},
	}
	credentialName := req.Provider + "-cred"

	cred := api.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:              credentialName,
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.CredentialSpec{
			Provider: req.Provider,
			Data:     make(map[string]string),
		},
	}
	credData := make(map[string]string)
	sb := api.StorageBackend{}
	switch req.Provider {
	case s3.Kind:
		if req.s3ConfigAccessKeyID != "" {
			credData[credential.AWSAccessKeyID] = req.s3ConfigAccessKeyID
		}
		if req.s3ConfigSecretKey != "" {
			credData[credential.AWSSecretAccessKey] = req.s3ConfigSecretKey
		}
		sb.S3 = &api.S3Spec{
			Endpoint: req.s3StoreEndpoint,
			Bucket:   req.s3StoreBucket,
			Prefix:   req.Prefix,
		}
	case gcs.Kind:
		if req.gcsConfigJSONKeyPath != "" {
			jsonKey, err := ioutil.ReadFile(req.gcsConfigJSONKeyPath)
			term.ExitOnError(err)
			credData[credential.GCEServiceAccount] = string(jsonKey)
		}
		if req.gcsConfigProjectId != "" {
			credData[credential.GCEProjectID] = req.gcsConfigProjectId
		}

		sb.GCS = &api.GCSSpec{
			Bucket: req.gcsStoreBucket,
			Prefix: req.Prefix,
		}
	case azure.Kind:
		if req.azureConfigAccount != "" {
			credData[credential.AzureStorageAccount] = req.azureConfigAccount
		}
		if req.azureConfigKey != "" {
			credData[credential.AzureStorageKey] = req.azureConfigKey
		}

		sb.Azure = &api.AzureStorageSpec{
			Container: req.azureStoreContainer,
			Prefix:    req.Prefix,
		}
	case local.Kind:
		sb.Local = &api.LocalSpec{
			Path: req.localConfigKeyPath,
		}
	case swift.Kind:
		// v2/v3 specific
		if req.swiftConfigUsername != "" {
			credData[credential.SwiftUsername] = req.swiftConfigUsername
		}
		if req.swiftConfigKey != "" {
			credData[credential.SwiftKey] = req.swiftConfigKey
		}
		if req.swiftConfigRegion != "" {
			credData[credential.SwiftRegion] = req.swiftConfigRegion
		}
		if req.swiftConfigTenantAuthURL != "" {
			credData[credential.SwiftTenantAuthURL] = req.swiftConfigTenantAuthURL
		}

		// v3
		if req.swiftConfigDomain != "" {
			credData[credential.SwiftDomain] = req.swiftConfigDomain
		}
		if req.swiftConfigTenantName != "" {
			credData[credential.SwiftTenantName] = req.swiftConfigTenantName
		}
		if req.swiftConfigTenantDomain != "" {
			credData[credential.SwiftTenantDomain] = req.swiftConfigTenantDomain
		}

		// v2 specific
		if req.swiftConfigTenantId != "" {
			credData[credential.SwiftTenantId] = req.swiftConfigTenantId
		}

		// v1 specific

		// Manual authentication
		if req.swiftConfigStorageURL != "" {
			credData[credential.SwiftStorageURL] = req.swiftConfigStorageURL
		}
		if req.swiftConfigAuthToken != "" {
			credData[credential.SwiftAuthToken] = req.swiftConfigAuthToken
		}

		sb.Swift = &api.SwiftSpec{
			Container: req.swiftStoreContainer,
			Prefix:    req.Prefix,
		}
	case "postgres":
		sb.Postgres = &api.PostgresSpec{
			DbName:   req.pgDatabaseName,
			Host:     req.pgHost,
			Port:     req.pgPort,
			User:     req.pgUser,
			Password: req.pgPassword,
		}
	default:
		term.Fatalln("Unknown provider:" + req.Provider)
	}

	if len(credData) != 0 {
		cred.Spec.Data = credData
		pc.Credentials = []api.Credential{cred}
		sb.CredentialName = credentialName
	}
	pc.Store = sb

	err := config.Save(pc, configPath)
	term.ExitOnError(err)
}

package system

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/appscode/go-dns/aws"
	"github.com/appscode/go-dns/azure"
	"github.com/appscode/go-dns/cloudflare"
	"github.com/appscode/go-dns/digitalocean"
	"github.com/appscode/go-dns/googlecloud"
	"github.com/appscode/go-dns/linode"
	"github.com/appscode/go-dns/vultr"
	"github.com/appscode/go-notify/mailgun"
	"github.com/appscode/go-notify/smtp"
	"github.com/appscode/go/encoding/yaml"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/strings"
	"github.com/appscode/log"
)

type URLBase struct {
	Scheme   string `json:"scheme,omitempty"`
	BaseAddr string `json:"base_addr,omitempty"`
}

type secureConfig struct {
	Secret               string `json:"secret,omitempty"`
	MagicCodeSecret      string `json:"magic_code_secret,omitempty"`
	SkipStartupConfigAPI bool   `json:"skip_startup_config_api,omitempty"`
	Database             struct {
		MetaNamespace string   `json:"meta_ns"`
		MetaHost      string   `json:"meta_host,omitempty"`
		Port          int      `json:"port,omitempty"`
		User          string   `json:"user,omitempty"`
		Password      string   `json:"password,omitempty"`
		Hosts         []string `json:"hosts,omitempty"`
	} `json:"database,omitempty"`
	Network struct {
		PublicUrls       URLBase `json:"public_urls,omitempty"`
		TeamUrls         URLBase `json:"team_urls,omitempty"`
		ClusterUrls      URLBase `json:"cluster_urls,omitempty"`
		InClusterUrls    URLBase `json:"in_cluster_urls,omitempty"`
		URLShortenerUrls URLBase `json:"URL_shortener_urls,omitempty"`
		FileUrls         URLBase `json:"file_urls,omitempty"`
	} `json:"network,omitempty"`
	Phabricator struct {
		Glusterfs struct {
			Endpoint string `json:"endpoint,omitempty"`
			Path     string `json:"path,omitempty"`
		} `json:"glusterfs"`
		DaemonImage            string `json:"daemon_image,omitempty"`
		PhabricatorDataProject string `json:"phabricator_data_project, omitempty"`
	} `json:"phabricator, omitempty"`
	Artifactory struct {
		ElasticSearchEndpoint string `json:"elasticsearch_endpoint,omitempty"`
	} `json:"artifactory,omitempty"`
	Compass struct {
		IPs []string `json:"ips,omitempty"`
	} `json:"compass,omitempty"`
	GoogleAnalytics struct {
		PublicTracker string `json:"public_tracker, omitempty"`
		TeamTracker   string `json:"team_tracker, omitempty"`
	} `json:"google_analytics, omitempty"`
	Twilio struct {
		Token       string `json:"token,omitempty"`
		AccountSid  string `json:"account_sid,omitempty"`
		PhoneNumber string `json:"phone_number,omitempty"`
	} `json:"twilio,omitempty"`
	Mail struct {
		PublicDomain string          `json:"public_domain,omitempty"`
		Mailer       string          `json:"mailer, omitempty"`
		Mailgun      mailgun.Options `json:"mailgun, omitempty"`
		SMTP         smtp.Options    `json:"smtp, omitempty"`
	} `json:"mail"`
	DNS struct {
		// Deprecated
		Credential     map[string]string `json:"credential,omitempty"`
		CredentialFile string            `json:"credential_file,omitempty"`

		// Generic DNS Providers
		Provider     string               `json:"provider,omitempty"`
		AWS          aws.Options          `json:"aws,omitempty"`
		Azure        azure.Options        `json:"azure,omitempty"`
		Cloudflare   cloudflare.Options   `json:"cloudflare,omitempty"`
		Digitalocean digitalocean.Options `json:"digitalocean,omitempty"`
		Gcloud       googlecloud.Options  `json:"gcloud,omitempty"`
		Linode       linode.Options       `json:"linode,omitempty"`
		Vultr        vultr.Options        `json:"vultr,omitempty"`
	} `json:"dns"`
	DigitalOcean struct {
		Token string `json:"token"`
	} `json:"digitalocean"`
	GCE struct {
		CredentialFile string `json:"credential_file,omitempty"`
	} `json:"gce"`
	S3 struct {
		AccessKey string `json:"access_key,omitempty"`
		SecretKey string `json:"secret_key,omitempty"`
		Region    string `json:"region,omitempty"`
		Endpoint  string `json:"endpoint,omitempty"`
	} `json:"s3"`
	Cowrypay struct {
		GRPCEndpoint string `json:"grpc_endpoint, omitempty"`
		HTTPEndpoint string `json:"http_endpoint, omitempty"`
		Namespace    string `json:"namespace, omitempty"`
		Token        string `json:"token, omitempty"`
	} `json:"cowrypay, omitempty"`
	Icinga2 struct {
		Host        string `json:"host"`
		APIUser     string `json:"api_user"`
		APIPassword string `json:"api_password"`
	} `json:"icinga2, omitempty"`
}

var Config secureConfig
var cOnce sync.Once

func Init() {
	cOnce.Do(func() {
		fmt.Println("[system] Reading system config file")
		env := _env.FromHost()
		configFiles := []string{
			"/srv/ark/config/config." + env.String() + ".yaml", // /srv/ark/config/config.env.yaml
			"/srv/ark/config/config." + env.String() + ".yml",  // /srv/ark/config/config.env.yml
			"/srv/ark/config/config." + env.String() + ".json", // /srv/ark/config/config.env.json
		}
		for _, cfgFile := range configFiles {
			fmt.Printf("Searching %v\n", cfgFile)
			if _, err := os.Stat(cfgFile); err == nil {
				data, err := ioutil.ReadFile(cfgFile)
				if err != nil {
					log.Fatal(err)
				}

				jsonData, err := yaml.ToJSON(data)
				if err != nil {
					log.Fatal(err)
				}

				err = json.Unmarshal(jsonData, &Config)
				if err != nil {
					log.Fatal(err)
				}
				applyDefaults(env)

				fmt.Println("[][][][][][][][][][][][][][][][][][][][][][][][][][][]")
				fmt.Printf("Using system configuration file %v\n", cfgFile)
				fmt.Println("[][][][][][][][][][][][][][][][][][][][][][][][][][][]")
				fmt.Println("******************************************************")
				return
			}
		}

		log.Fatalln("Missing system configuration file.")
	})
}

func applyDefaults(env _env.Environment) {
	if env == _env.Prod {
		Config.Network.PublicUrls.BaseAddr = "appscode.com"
		Config.Network.TeamUrls.BaseAddr = "appscode.io"
		Config.Network.ClusterUrls.BaseAddr = "containercloud.io"
		Config.Network.URLShortenerUrls.BaseAddr = "appsco.de"
		Config.Network.FileUrls.BaseAddr = "appscode.space"
	} else if env == _env.QA {
		Config.Network.PublicUrls.BaseAddr = "appscode.info"
		Config.Network.TeamUrls.BaseAddr = "appscode.ninja"
		Config.Network.ClusterUrls.BaseAddr = "containercloud.xyz"
		Config.Network.URLShortenerUrls.BaseAddr = "appscode.co"
		Config.Network.FileUrls.BaseAddr = "appscode.org"
	} else if env == _env.Dev {
		Config.Network.PublicUrls.BaseAddr = strings.Val(Config.Network.PublicUrls.BaseAddr, "appscode.dev")
		Config.Network.TeamUrls.BaseAddr = strings.Val(Config.Network.TeamUrls.BaseAddr, "appscode.dev")
		Config.Network.ClusterUrls.BaseAddr = strings.Val(Config.Network.ClusterUrls.BaseAddr, "containercloud.xyz")
		Config.Network.URLShortenerUrls.BaseAddr = strings.Val(Config.Network.URLShortenerUrls.BaseAddr, "appscode.co")
		Config.Network.FileUrls.BaseAddr = strings.Val(Config.Network.FileUrls.BaseAddr, "appscode.org")
	} else if !env.IsHosted() {
		// TeamUrls.BaseDomain must be provided by user
		// Config.Network.TeamUrls.BaseDomain
		Config.Network.PublicUrls.BaseAddr = Config.Network.TeamUrls.BaseAddr
		Config.Network.ClusterUrls.BaseAddr = "kubernetes." + Config.Network.TeamUrls.BaseAddr
		Config.Network.URLShortenerUrls.BaseAddr = "x." + Config.Network.TeamUrls.BaseAddr
		Config.Network.FileUrls.BaseAddr = Config.Network.TeamUrls.BaseAddr
	}
	Config.Network.InClusterUrls.Scheme = "http"
	Config.Network.InClusterUrls.BaseAddr = "default"
}

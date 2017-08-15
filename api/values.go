package api

const (
	CertTrusted = iota - 1
	CertRoot
	CertNSRoot
	CertIntermediate
	CertLeaf

	RoleJenkinsMaster = "jenkins-master"
	RoleJenkinsAgent  = "jenkins-agent"

	RoleKubernetesMaster = "kubernetes-master"
	RoleKubernetesPool   = "kubernetes-pool"

	CIBotUser      = "ci-bot"
	ClusterBotUser = "k8s-bot"

	PhabricatorShowPrototypes      = "phabricator.show-prototypes"
	AllowHttpAuth                  = "diffusion.allow-http-auth"
	AppsCodePrivateAPIHttpEndpoint = "appscode.private-api-http-endpoint"
	AppsCodePublicAPIHttpEndpoint  = "appscode.public-api-http-endpoint"
	CowrypayCustomerID             = "cowrypay.customer-id"
	CIProvider                     = "ci.provider"
	CIDefaultBot                   = "ci.default-bot"
	CSRF_Key                       = "phabricator.csrf-key"
	CSRF_Value                     = "0b7ec0592e0a2829d8b71df2fa269b2c6172eca3"
	DigitalOceanCredential         = "digitalocean.credential"
	DNSCredential                  = "dns.credential"
	ElasticSearchHost              = "search.elastic.host"
	ElasticSearchNamespace         = "search.elastic.namespace"
	ArtifactoryES                  = "artifactory.elastic.host"
	MailgunApiKey                  = "mailgun.api-key"
	MailgunPublicDomain            = "mailgun.public-domain"
	MailgunTeamDomain              = "mailgun.domain"
	SMTPPublicDomain               = "smtp.public-domain"
	SMTPHost                       = "phpmailer.smtp-host"
	SMTPPassword                   = "phpmailer.smtp-password"
	SMTPPort                       = "phpmailer.smtp-port"
	SMTPUser                       = "phpmailer.smtp-user"
	MetamtaDefaultAddress          = "metamta.default-address"
	MetamtaDomain                  = "metamta.domain"
	MetamtaMailAdapter             = "metamta.mail-adapter"
	MetamtaReplyHandlerDomain      = "metamta.reply-handler-domain"
	NSDeactivationPeriod           = "ns.deactivation-period"
	NSMinRollingPayment            = "ns.min-rolling-payment"
	NSRollingPayment               = "ns.rolling-payment"
	PhabricatorBucket              = "phabricator.data-bucket-name"
	PhabricatorS3Bucket            = "storage.s3.bucket"
	PhabricatorLocalDiskStorage    = "storage.local-disk.path"
	PygmentsEnabled                = "pygments.enabled"
	RepositoryPath                 = "repository.default-local-path"
	SecurityAlternateFileDomain    = "security.alternate-file-domain"
	PhabricatorNotificationServer  = "notification.servers"
	ShortUri                       = "phurl.short-uri"
	TwilioAccountSid               = "twilio.account-sid"
	TwilioAuthToken                = "twilio.auth-token"
	TwilioPhoneNumber              = "twilio.phone-number"
	VCSHost                        = "diffusion.ssh-host"
	VCSUser                        = "diffusion.ssh-user"
	GCS                            = "gcs"
	S3                             = "s3"
	Local                          = "local"
	Mailgun                        = "mailgun"
	SMTP                           = "smtp"
	PhabricatorProfileImageKeyURI  = "user.profile.image.uri.v1"
)

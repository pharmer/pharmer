package api

const (
	CertTrusted = iota - 1
	CertRoot
	CertNSRoot
	CertIntermediate
	CertLeaf

	RoleKubernetesMaster = "kubernetes-master"
	RoleKubernetesPool   = "kubernetes-pool"

	CIBotUser      = "ci-bot"
	ClusterBotUser = "k8s-bot"

	CSRF_Key                    = "phabricator.csrf-key"
	CSRF_Value                  = "0b7ec0592e0a2829d8b71df2fa269b2c6172eca3"
	DigitalOceanCredential      = "digitalocean.credential"
	DNSCredential               = "dns.credential"
	ArtifactoryES               = "artifactory.elastic.host"
	MailgunApiKey               = "mailgun.api-key"
	MailgunPublicDomain         = "mailgun.public-domain"
	SMTPPassword                = "phpmailer.smtp-password"
	MetamtaDefaultAddress       = "metamta.default-address"
	MetamtaDomain               = "metamta.domain"
	MetamtaReplyHandlerDomain   = "metamta.reply-handler-domain"
	NSDeactivationPeriod        = "ns.deactivation-period"
	NSMinRollingPayment         = "ns.min-rolling-payment"
	NSRollingPayment            = "ns.rolling-payment"
	RepositoryPath              = "repository.default-local-path"
	SecurityAlternateFileDomain = "security.alternate-file-domain"
	VCSHost                     = "diffusion.ssh-host"
	GCS                         = "gcs"
	S3                          = "s3"
	Local                       = "local"
	Mailgun                     = "mailgun"
	SMTP                        = "smtp"
)

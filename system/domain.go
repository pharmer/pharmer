package system

import (
	"crypto/md5"
	"encoding/base32"
	"fmt"
	"strings"

	_env "github.com/appscode/go/env"
)

func Scheme(ub URLBase) string {
	if ub.Scheme != "" {
		return ub.Scheme
	} else {
		if _env.FromHost().DevMode() {
			return "http"
		} else {
			return "https"
		}
	}
}

func PublicBaseDomain() string {
	return strings.SplitN(Config.Network.PublicUrls.BaseAddr, ":", 2)[0]
}

/*
apiserver ports - 50077 (public)   -> 3443 (https://)
                - 50099 (private)
proxy           - 9877  (public)   -> 443  (https://)
                - 9899  (private)
*/
func APIAddr() string {
	return "api." + Config.Network.PublicUrls.BaseAddr
}

// https://api.appscode.com:443
func PublicAPIHttpEndpoint() string {
	return Scheme(Config.Network.PublicUrls) + "://" + APIAddr()
}

// https://api.appscode.com:3443
func PublicAPIGrpcEndpoint() string {
	addr := APIAddr()
	return Scheme(Config.Network.PublicUrls) + "://" + strings.SplitN(addr, ":", 2)[0] + ":3443"
}

func PublicAPIHttpURL(trails ...string) string {
	return strings.TrimRight(PublicAPIHttpEndpoint()+"/"+strings.Join(trails, "/"), "/")
}

func KuberntesWebhookAuthenticationURL() string {
	return PublicAPIHttpURL("kubernetes/v1beta1/webhooks/authentication")
}

func KuberntesWebhookAuthorizationURL() string {
	return PublicAPIHttpURL("kubernetes/v1beta1/webhooks/authorization")
}

func InClusterPublicAPIHttpEndpoint() string {
	baseDomain := strings.SplitN(Config.Network.InClusterUrls.BaseAddr, ":", 2)[0]
	return Scheme(Config.Network.InClusterUrls) + "://apiserver." + baseDomain + ":9877"
}

func InClusterPrivateAPIHttpEndpoint() string {
	baseDomain := strings.SplitN(Config.Network.InClusterUrls.BaseAddr, ":", 2)[0]
	return Scheme(Config.Network.InClusterUrls) + "://apiserver." + baseDomain + ":9899"
}

func DockerAddr() string {
	return "docker." + Config.Network.PublicUrls.BaseAddr
}

func MavenAddr() string {
	return "maven." + Config.Network.PublicUrls.BaseAddr
}

func DockerURL() string {
	return Scheme(Config.Network.PublicUrls) + "://" + DockerAddr()
}

func SubDomain(ns string) string {
	return ns
}

func TeamAddr(ns string) string {
	if _env.FromHost().IsHosted() {
		return SubDomain(ns) + "." + Config.Network.TeamUrls.BaseAddr
	} else {
		return Config.Network.TeamUrls.BaseAddr
	}
}

func TeamDomain(ns string) string {
	return strings.SplitN(TeamAddr(ns), ":", 2)[0]
}

func TeamRootURL(ns string) string {
	return Scheme(Config.Network.TeamUrls) + "://" + TeamAddr(ns)
}

func TeamURL(ns string, trails ...string) string {
	return strings.TrimRight(TeamRootURL(ns)+"/"+strings.Join(trails, "/"), "/")
}

func ClusterBaseDomain() string {
	return strings.SplitN(Config.Network.ClusterUrls.BaseAddr, ":", 2)[0]
}

func ClusterAddr(ns, cluster string) string {
	if _env.FromHost().IsHosted() {
		return strings.ToLower(cluster) + "-" + SubDomain(ns) + "." + Config.Network.ClusterUrls.BaseAddr
	} else {
		return strings.ToLower(cluster) + "." + Config.Network.ClusterUrls.BaseAddr
	}
}

func ClusterDomain(ns, cluster string) string {
	return strings.SplitN(ClusterAddr(ns, cluster), ":", 2)[0]
}

func ClusterRootURL(ns, cluster string) string {
	return Scheme(Config.Network.ClusterUrls) + "://" + ClusterExternalDomain(ns, cluster)
}

func ClusterURL(ns, cluster string, trails ...string) string {
	return strings.TrimRight(ClusterRootURL(ns, cluster)+"/"+strings.Join(trails, "/"), "/")
}

func ClusterExternalDomain(ns, cluster string) string {
	return "api." + ClusterDomain(ns, cluster)
}

func ClusterInternalDomain(ns, cluster string) string {
	return "internal." + ClusterDomain(ns, cluster)
}

func ClusterCAName(ns, cluster string) string {
	return "ca@" + ClusterDomain(ns, cluster)
}

func ClusterUsername(ns, cluster, user string) string {
	return user + "@" + ClusterDomain(ns, cluster)
}

func FileDomain(ns string) string {
	return SubDomain(ns) + "." + Config.Network.FileUrls.BaseAddr
}

func FileURL(ns string) string {
	return Scheme(Config.Network.FileUrls) + "://" + FileDomain(ns) + "/"
}

func GitHostingDomain() string {
	return "diffusion." + Config.Network.PublicUrls.BaseAddr
}

func URLShortenerDomain(ns string) string {
	return SubDomain(ns) + "." + Config.Network.URLShortenerUrls.BaseAddr
}

func URLShortenerUrl(ns string) string {
	return Scheme(Config.Network.URLShortenerUrls) + "://" + URLShortenerDomain(ns) + "/"
}

func MailgunInboundURL(ns string) string {
	if _env.FromHost().IsHosted() {
		// https://\g<ns>.appscode.io/mail/mailgun/
		return Scheme(Config.Network.TeamUrls) + `://\g<ns>.` + Config.Network.TeamUrls.BaseAddr + "/mail/mailgun/"
	} else {
		// https://getappscode.com/mail/mailgun/
		return Scheme(Config.Network.TeamUrls) + `://` + Config.Network.TeamUrls.BaseAddr + "/mail/mailgun/"
	}
}

func MailgunRecipientRegex(ns string) string {
	baseDomain := strings.SplitN(Config.Network.TeamUrls.BaseAddr, ":", 2)[0]
	if _env.FromHost().IsHosted() {
		// ^[a-zA-Z0-9_.+-]+@(?P<ns>[a-zA-Z0-9-]+)\.appscode\.io+$
		return fmt.Sprintf(`^[a-zA-Z0-9_.+-]+@(?P<ns>[a-zA-Z0-9-]+)\.%v+$`, strings.Replace(baseDomain, `.`, `\.`, -1))
	} else {
		// ^[a-zA-Z0-9_.+-]+@getappscode\.com+$
		return fmt.Sprintf(`^[a-zA-Z0-9_.+-]+@%v+$`, strings.Replace(baseDomain, `.`, `\.`, -1))
	}
}

func MailAdapter() string {
	return "PhabricatorMailImplementationMailgunAdapter"
}

func MailDefaultAddress(ns string) string {
	return "noreply@" + TeamAddr(ns)
}

func GrafanaEndpoint(ns, cluster string) string {
	return ClusterURL(ns, cluster, "grafana")
}

//func GraphanaClusterURL(ns, cluster, dashboardName string) string {
//	return GrafanaEndpoint(ns, cluster) + "/dashboard/db/" + dashboardName
//}
//
//func GraphanaPodURL(ns, cluster, kubeNamespace, podName string) string {
//	return GraphanaClusterURL(ns, cluster, "pods") + fmt.Sprintf("?var-namespace=%s&var-podname=%s", kubeNamespace, podName)
//}
//
//func GraphanaServiceURL(ns, cluster, kubeNamespace, serviceName string) string {
//	return GraphanaClusterURL(ns, cluster, "services") + fmt.Sprintf("?var-namespace=%s&var-service=%s", kubeNamespace, serviceName)
//}

func IcingaWebEndpoint(ns, cluster string) string {
	return ClusterURL(ns, cluster, "icingaweb2")
}

func IcingaWebDashboard(ns, cluster string) string {
	return IcingaWebEndpoint(ns, cluster) + "/dashboard"
}

func IcingaWebServiceURL(ns, cluster, icingaHost, icingService string) string {
	return IcingaWebEndpoint(ns, cluster) + fmt.Sprintf(`/monitoring/service/show?host=%s&service=%s`, icingaHost, icingService)
}

//func IcingaWebAlertURL(ns, cluster, alertPhid string) string {
//	return IcingaWebEndpoint(ns, cluster) + fmt.Sprintf(`/monitoring/list/hosts?_service_alert_phid=%s#!/icingaweb2/monitoring/list/services?_service_alert_phid=%s`, alertPhid, alertPhid)
//}
//
//func IcingaWebHostUrl(ns, cluster, icingaHost string) string {
//	return IcingaWebEndpoint(ns, cluster) + fmt.Sprintf(`/monitoring/list/hosts?host=%s#!/icingaweb2/monitoring/list/services?host=%s`, icingaHost, icingaHost)
//}
//
//func IcingaWebAppURL(ns, cluster, appFilter, appName string) string {
//	return IcingaWebEndpoint(ns, cluster) + fmt.Sprintf(`/monitoring/list/hosts?%s=true&sort=host_state&dir=desc#!/icingaweb2/monitoring/list/services?%s=true&_service_app_name=%s&sort=service_state&dir=desc`, appFilter, appFilter, appName)
//}

func IcingaWebIncidentUrl(ns, cluster, hostName, serviceName string) string {
	return IcingaWebEndpoint(ns, cluster) + fmt.Sprintf(`/monitoring/list/hosts?host=%s#!/icingaweb2/monitoring/list/services?host=%s&service=%s`, hostName, hostName, serviceName)
}

//func IcingaWebAppHostURL(ns, cluster, hostName, appName string) string {
//	return IcingaWebEndpoint(ns, cluster) + fmt.Sprintf(`/monitoring/list/hosts?host=%s#!/icingaweb2/monitoring/list/services?host=%s&_service_app_name=%s`, hostName, hostName, appName)
//}

func KibanaEndpoint(ns, cluster string) string {
	return ClusterURL(ns, cluster, "kibana")
}

//func KibanaPodUrl(ns, cluster, namespace, podName string) string {
//	return KibanaEndpoint(ns, cluster) + fmt.Sprintf(`/app/kibana#/discover?_a=(columns:!(kubernetes.container_name,log),index:'logstash-*',query:(query_string:(query:'kubernetes.namespace_name:"%s" AND kubernetes.pod_name:"%s"')))&_g=(time:(from:now-6h,mode:quick,to:now))`, namespace, podName)
//}

func PhabricatorDataBucket(ns string) string {
	return databucket(ns, "phabricator")
}

func databucket(ns, app string) string {
	ns = strings.ToLower(ns)
	h := md5.New()
	h.Write([]byte(ns))
	h.Write([]byte(":"))
	h.Write([]byte(_env.FromHost().String()))
	hash := base32.StdEncoding.EncodeToString(h.Sum(nil))
	name := fmt.Sprintf("%s-%s-data-", ns, app)
	if len(name) > 54 {
		name = name[:54]
	}
	return strings.ToLower(name + hash[0:10]) // max length = 64
}

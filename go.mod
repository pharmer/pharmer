module pharmer.dev/pharmer

go 1.12

require (
	github.com/Azure/azure-sdk-for-go v29.0.0+incompatible
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v11.1.2+incompatible
	github.com/Netflix/go-expect v0.0.0-20180928190340-9d1f4485533b // indirect
	github.com/appscode/go v0.0.0-20190621064509-6b292c9166e3
	github.com/aws/aws-sdk-go v1.19.31
	github.com/creack/goselect v0.1.0 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20190515213511-eb9f6a1743f3 // indirect
	github.com/digitalocean/godo v1.14.0
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/jsonpointer v0.19.0 // indirect
	github.com/go-openapi/jsonreference v0.19.0 // indirect
	github.com/go-openapi/swag v0.19.0 // indirect
	github.com/go-xorm/core v0.6.2
	github.com/go-xorm/xorm v0.7.3
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.1.1
	github.com/gophercloud/gophercloud v0.0.0-20190515011819-1992d5238d78 // indirect
	github.com/graymeta/stow v0.0.0-00010101000000-000000000000
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/jpillora/go-ogle-analytics v0.0.0-20161213085824-14b04e0594ef
	github.com/kr/pty v1.1.4 // indirect
	github.com/lib/pq v1.1.1
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/linode/linodego v0.8.0
	github.com/mailru/easyjson v0.0.0-20190403194419-1ea4449da983 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/nats-io/stan.go v0.5.0
	github.com/ncw/swift v1.0.47 // indirect
	github.com/onsi/gomega v1.5.0
	github.com/packethost/packngo v0.1.1-0.20190507131943-1343be729ca2
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.4 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spf13/cobra v0.0.4
	github.com/spf13/pflag v1.0.3
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8
	golang.org/x/net v0.0.0-20190613194153-d28f0bde5980 // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	golang.org/x/sys v0.0.0-20190613124609-5ed2794edfdc // indirect
	gomodules.xyz/cert v1.0.0
	gomodules.xyz/nats-logr v0.1.0
	gomodules.xyz/union-logr v0.1.0
	gomodules.xyz/version v0.0.0-20190507203204-7cec7ee542d3
	google.golang.org/api v0.5.0
	google.golang.org/appengine v1.6.1 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.6.1
	gopkg.in/ini.v1 v1.42.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190531132109-d3f5f50bdd94
	k8s.io/apiextensions-apiserver v0.0.0-20190515024537-2fd0e9006049
	k8s.io/apimachinery v0.0.0-20190531131812-859a0ba5e71a
	k8s.io/cli-runtime v0.0.0-20190515024640-178667528169
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cluster-bootstrap v0.0.0-20181204005900-2d1c733eadd0
	k8s.io/component-base v0.0.0-20190515024022-2354f2393ad4 // indirect
	k8s.io/klog v0.3.2
	k8s.io/kubernetes v1.14.2
	k8s.io/sample-controller v0.0.0-20190531134801-325dc0a18ed9
	kmodules.xyz/client-go v0.0.0-20190524133821-9c8a87771aea
	pharmer.dev/cloud v0.3.0
	sigs.k8s.io/cluster-api v0.0.0-20190508175234-0f911c1f65a5
	sigs.k8s.io/controller-runtime v0.2.0-beta.4
)

replace (
	github.com/graymeta/stow => github.com/appscode/stow v0.0.0-20190506085026-ca5baa008ea3
	github.com/renstrom/fuzzysearch => github.com/lithammer/fuzzysearch v1.0.1-0.20160331204855-2d205ac6ec17
	gopkg.in/robfig/cron.v2 => github.com/appscode/cron v0.0.0-20170717094345-ca60c6d796d4
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed
	k8s.io/apimachinery => github.com/kmodules/apimachinery v0.0.0-20190508045248-a52a97a7a2bf
	k8s.io/apiserver => github.com/kmodules/apiserver v0.0.0-20190508082252-8397d761d4b5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190314001948-2899ed30580f
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190314002645-c892ea32361a
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190314000054-4a91899592f4
	k8s.io/klog => k8s.io/klog v0.3.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190314000639-da8327669ac5
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190314001731-1bd6a4002213
	k8s.io/utils => k8s.io/utils v0.0.0-20190221042446-c2654d5206da
	sigs.k8s.io/cluster-api => github.com/pharmer/cluster-api v0.0.0-20190516113055-efd9b156f6d2
)

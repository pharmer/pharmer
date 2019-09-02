module pharmer.dev/pharmer

go 1.12

require (
	github.com/Azure/azure-sdk-for-go v31.1.0+incompatible
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest/autorest v0.5.0
	github.com/Azure/go-autorest/autorest/adal v0.2.0
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/Netflix/go-expect v0.0.0-20180928190340-9d1f4485533b // indirect
	github.com/appscode/go v0.0.0-20190722173419-e454bf744023
	github.com/aws/aws-sdk-go v1.20.20
	github.com/creack/goselect v0.1.0 // indirect
	github.com/digitalocean/godo v1.14.0
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/jsonpointer v0.19.0 // indirect
	github.com/go-openapi/jsonreference v0.19.0 // indirect
	github.com/go-openapi/swag v0.19.0 // indirect
	github.com/go-xorm/xorm v0.7.4
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.1.1
	github.com/gophercloud/gophercloud v0.0.0-20190515011819-1992d5238d78 // indirect
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/jpillora/go-ogle-analytics v0.0.0-20161213085824-14b04e0594ef
	github.com/kr/pty v1.1.4 // indirect
	github.com/lib/pq v1.2.0
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/linode/linodego v0.8.0
	github.com/mailru/easyjson v0.0.0-20190403194419-1ea4449da983 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/nats-io/stan.go v0.5.0
	github.com/onsi/gomega v1.5.0
	github.com/packethost/packngo v0.1.1-0.20190507131943-1343be729ca2
	github.com/pkg/errors v0.8.1
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	go.etcd.io/bbolt v1.3.3 // indirect
	gocloud.dev v0.16.0
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	gomodules.xyz/cert v1.0.0
	gomodules.xyz/nats-logr v0.1.0
	gomodules.xyz/secrets v0.2.2-0.20190902103609-7fbbd02d7e9d
	gomodules.xyz/stow v0.2.0
	gomodules.xyz/union-logr v0.1.0
	gomodules.xyz/version v0.0.0-20190507203204-7cec7ee542d3
	google.golang.org/api v0.7.0
	gopkg.in/AlecAivazis/survey.v1 v1.6.1
	gopkg.in/ini.v1 v1.42.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190531132109-d3f5f50bdd94
	k8s.io/apiextensions-apiserver v0.0.0-20190516231611-bf6753f2aa24
	k8s.io/apimachinery v0.0.0-20190531131812-859a0ba5e71a
	k8s.io/cli-runtime v0.0.0-20190516231937-17bc0b7fcef5
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cluster-bootstrap v0.0.0-20181204005900-2d1c733eadd0
	k8s.io/component-base v0.0.0-20190515024022-2354f2393ad4 // indirect
	k8s.io/klog v0.3.2
	k8s.io/kubernetes v1.14.2
	kmodules.xyz/client-go v0.0.0-20190715080709-7162a6c90b04
	pharmer.dev/cloud v0.3.0
	sigs.k8s.io/cluster-api v0.0.0-20190508175234-0f911c1f65a5
	sigs.k8s.io/controller-runtime v0.2.0-beta.4
	xorm.io/core v0.6.3
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest/autorest v0.5.0
	github.com/renstrom/fuzzysearch => github.com/lithammer/fuzzysearch v1.0.1-0.20160331204855-2d205ac6ec17
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

package db

import "time"

type Certificate struct {
	Id            int64
	PHID          string    `xorm:"text not null 'phid'" mapper:"target=Phid"`
	Name          string    `xorm:"text not null 'name'" mapper:"target=Name"`
	CommonName    string    `xorm:"text NOT NULL 'commonName'" mapper:"target=CommonName"`
	SANs          string    `xorm:"text 'sans'"`
	Domain        string    `xorm:"text 'domain'"`
	CertURL       string    `xorm:"text 'certUrl'"`
	CertStableURL string    `xorm:"text 'certStableUrl'"`
	AccountPHID   string    `xorm:"text 'accountPHID'"`
	IssuedBy      string    `xorm:"text NOT NULL 'issuedBy'" mapper:"target=IssuedBy"`
	Cert          string    `xorm:"text NOT NULL 'cert'" mapper:"target=Cert"`
	Key           string    `xorm:"text NOT NULL 'key'"`
	Serial        string    `xorm:"text NOT NULL 'serial'" mapper:"target=SerialNumber"`
	Version       int32     `xorm:"bigint not null 'version'" mapper:"target=Version"`
	CertType      int8      `xorm:"smallint not null 'certType'" mapper:"-"`
	IsRevoked     int8      `xorm:"smallint not null 'isRevoked'" mapper:"-"`
	Reason        int8      `xorm:"smallint  'reason'" mapper:"-"`
	Status        string    `xorm:"text 'status'" mapper:"-"`
	ValidFrom     time.Time `xorm:"bigint not null 'validFrom'" mapper:"-"`
	ExpireDate    time.Time `xorm:"bigint not null 'expireDate'" mapper:"-"`
	IsDeleted     int8      `xorm:"smallint not null 'isDeleted'" mapper:"-"`
	DateCreated   time.Time `xorm:"bigint created 'dateCreated'" mapper:"-"`
	DateModified  time.Time `xorm:"bigint updated 'dateModified'" mapper:"-"`
}

func (t Certificate) TableName() string {
	return `"ac_certificate"."certificate"`
}

type CloudCredential struct {
	Id           int64
	PHID         string    `xorm:"text not null 'phid'"`
	Name         string    `xorm:"text not null 'name'"`
	UserName     string    `xorm:"text not null 'userName'"`
	Provider     string    `xorm:"text not null 'provider'"`
	Data         string    `xorm:"text not null 'data'"`
	DateCreated  time.Time `xorm:"bigint created 'dateCreated'"`
	DateModified time.Time `xorm:"bigint updated 'dateModified'"`
}

func (t CloudCredential) TableName() string {
	return `"ac_cluster"."cloud_credential"`
}

type SSHKey struct {
	Id                 int64
	PHID               string    `xorm:"string  not null 'phid'"`
	Name               string    `xorm:"text  not null 'name'"`
	PublicKey          string    `xorm:"string  not null 'publicKey'"`
	PrivateKey         string    `xorm:"string  not null 'privateKey'"`
	AWSFingerprint     string    `xorm:"string  not null 'awsFingerprint'"`
	OpenSSHFingerprint string    `xorm:"string  not null 'opensshFingerprint'"`
	IsDeleted          int32     `xorm:"smallint not null 'isDeleted'"`
	DateCreated        time.Time `xorm:"bigint created 'dateCreated'"`
	DateModified       time.Time `xorm:"bigint updated 'dateModified'"`
}

func (t SSHKey) TableName() string {
	return `"ac_cluster"."ssh_key"`
}

type ClusterOP int64

type ClusterEdge struct {
	Src     string    `xorm:"text not null 'src'"`
	Type    ClusterOP `xorm:"bigint not null 'type'"`
	Dst     string    `xorm:"text not null 'dst'"`
	Created time.Time `xorm:"bigint created 'dateCreated'"`
	Seq     int64     `xorm:"bigint not null 'seq'"`
	DataID  int64     `xorm:"bigint 'dataID'"`
}

func (e ClusterEdge) TableName() string {
	return `"ac_cluster"."edge"`
}

type Kubernetes struct {
	ID                     int64     `xorm:"bigint not null 'id'" mapper:"-"`
	PHID                   string    `xorm:"text not null 'phid'" mapper:"target=Phid"`
	Name                   string    `xorm:"text not null 'name'" mapper:"target=Name"`
	Provider               string    `xorm:"text not null 'provider'" mapper:"target=Provider"`
	ProviderCredentialPHID string    `xorm:"text not null 'providerCredentialPHID'" mapper:"-"`
	Region                 string    `xorm:"text not null 'region'" mapper:"target=Region"`
	Zone                   string    `xorm:"text not null 'zone'" mapper:"target=Zone"`
	OS                     string    `xorm:"text not null 'os'" mapper:"target=Os"`
	CACertPHID             string    `xorm:"text not null 'cACertPHID'" mapper:"-"`
	SSHKeyPHID             string    `xorm:"text not null 'sshKeyPHID'" mapper:"-"`
	ApiServerURL           string    `xorm:"text not null 'apiServerURL'" mapper:"target=ApiServerUrl"`
	Status                 string    `xorm:"text not null 'status'" mapper:"target=Status"`
	StatusCause            string    `xorm:"text 'statusCause'" mapper:"target=StatusCause"`
	BucketName             string    `xorm:"text not null 'bucketName'" mapper:"-"`
	ContextVersion         int64     `xorm:"text not null 'contextVersion'" mapper:"-"`
	StartupConfigToken     string    `xorm:"text not null 'startupConfigToken'" mapper:"-"`
	DoNotDelete            int32     `xorm:"smallint not null default 0 'doNotDelete'"`
	DefaultAccessLevel     string    `xorm:"text not null 'defaultAccessLevel'" mapper:"-"`
	DateCreated            time.Time `xorm:"bigint created 'dateCreated'" mapper:"-"`
	DateModified           time.Time `xorm:"bigint updated 'dateModified'" mapper:"-"`
}

func (t Kubernetes) TableName() string {
	return `"ac_cluster"."kubernetes"`
}

type KubernetesVersion struct {
	ID           int64     `xorm:"bigint not null 'id'"`
	ClusterName  string    `xorm:"text 'clusterName'"`
	Context      string    `xorm:"text 'context'" mapper:"-"`
	DateCreated  time.Time `xorm:"bigint created 'dateCreated'" mapper:"-"`
	DateModified time.Time `xorm:"bigint updated 'dateModified'" mapper:"-"`
}

func (t KubernetesVersion) TableName() string {
	return `"ac_cluster"."kubernetes_version"`
}

type S_KubernetesInstance struct {
	ID             int64     `xorm:"bigint not null 'id'"`
	PHID           string    `xorm:"text not null 'phid'"`
	KubernetesPHID string    `xorm:"text not null 'kubernetesPHID'"`
	ExternalID     string    `xorm:"text not null 'externalID'"`
	ExternalStatus string    `xorm:"text not null 'externalStatus'"`
	Name           string    `xorm:"text not null 'name'"`
	ExternalIP     string    `xorm:"text 'externalIP'"`
	InternalIP     string    `xorm:"text 'internalIP'"`
	SKU            string    `xorm:"text not null 'sku'"`
	Role           string    `xorm:"text not null 'role'"`
	Status         string    `xorm:"text not null 'status'"`
	DateCreated    time.Time `xorm:"bigint created 'dateCreated'"`
	DateModified   time.Time `xorm:"bigint updated 'dateModified'"`
}

func (t S_KubernetesInstance) TableName() string {
	return `"ac_cluster"."kubernetes_instance"`
}

package api

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

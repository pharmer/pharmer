package storage

import (
	"time"
)

type User struct {
	Id                 int64
	Phid               string    `xorm:"text not null 'phid'"`
	UserName           string    `xorm:"varchar(128) not null 'userName'"`
	RealName           string    `xorm:"varchar(256) not null 'realName'"`
	PasswordSalt       string    `xorm:"varchar(64)  not null 'passwordSalt'"`
	PasswordHash       string    `xorm:"varchar(256) not null 'passwordHash'"`
	ConduitCertificate string    `xorm:"varchar(510) not null 'conduitCertificate'"`
	IsSystemAgent      int8      `xorm:"smallint    'isSystemAgent'"`
	IsDisabled         int8      `xorm:"smallint     not null 'isDisabled'"`
	IsAdmin            int8      `xorm:"smallint     not null 'isAdmin'"`
	IsEmailVerified    int64     `xorm:"bigint       not null 'isEmailVerified'"`
	IsApproved         int64     `xorm:"bigint       not null 'isApproved'"`
	AccountSecret      string    `xorm:"text not null 'accountSecret'"`
	IsMailingList      int8      `xorm:"smallint not null 'isMailingList'"`
	DateCreated        time.Time `xorm:"bigint created 'dateCreated'"`
	DateModified       time.Time `xorm:"bigint updated 'dateModified'"`
}

func (u User) TableName() string {
	return `"user"."user"`
}

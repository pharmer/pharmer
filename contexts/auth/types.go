package auth

import (
	"github.com/appscode/pharmer/storage"
)

type Provider string

const (
	Gearman Provider = "GEARMAN"
	Server  Provider = "SERVER"
)

type AuthType int

const (
	LoggedVia_Basic  AuthType = iota // 0
	LoggedVia_Bearer                 // 1
	LoggedVia_Cookie                 // 2
)

const (
	Authentication     string = "authentication"
	CookieKeyToken     string = "phtkn"
	CookieKeySessionID string = "phsid"
)

type AuthInfo struct {
	// Provider defines the provider from which the
	// auth is initiated.
	Provider  Provider
	Namespace string

	// Secret Contains the secret provided. If AuthType is
	// Bearer this contains the token.
	Token string

	// LoggedVia indicates is this object was obtained from
	// request cookies. used only in web requests.
	LoggedVia AuthType
	SessionID string

	// The authenticated user object obtained from the phabricator database.
	User storage.User
}

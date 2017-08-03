package phid

import (
	"strings"

	"github.com/appscode/go/crypto/rand"
)

func new(prefix, t, s string) string {
	if s == "" {
		return prefix + "-" + t + "-" + rand.Characters(20)
	} else {
		return prefix + "-" + t + "-" + s + "-" + rand.Characters(15)
	}
}

func newACID(t, s string) string {
	return new("ACID", t, s)
}

func newPHID(t, s string) string {
	return new("PHID", t, s)
}

func IsPHID(s string) bool {
	// ignores system secret
	// TODO(admin): Should we ignore case sensitivity?
	return strings.HasPrefix(s, "PHID-") || strings.HasPrefix(s, "ACID-")
}

// --- Special Constants ---
func SystemSecret() string {
	return "SYS-SCRT-PHID" // never conflicts with other *IDs
}

// --- Phabricator Defined PHIDs ---
func NewUser() string {
	return newPHID("USER", "")
}

func NewUserPref() string {
	return newPHID("USER", "PREF")
}

func NewUserExternal() string {
	return newPHID("XUSR", "")
}

func NewConfig() string {
	return newPHID("CONF", "")
}

func NewAuthProvider() string {
	return newPHID("AUTH", "")
}

func NewDashboard() string {
	return newPHID("DSHB", "")
}

func NewDashboardTransaction() string {
	return newPHID("XACT", "DSHB")
}

func NewDashboardPanel() string {
	return newPHID("DSHP", "")
}
func NewDashboardPanelTransaction() string {
	return newPHID("XACT", "DSHP")
}

func NewHomeApplication() string {
	return newPHID("APPS", "")
}

func NewProfileMenuItem() string {
	return newPHID("PANL", "")
}

func NewRUri() string {
	return newPHID("RURI", "")
}

func NewCalanderEvent() string {
	return newPHID("CEVT", "")
}

func NewConpherenceThread() string {
	return newPHID("CONP", "")
}

func NewDocumentField() string {
	return newPHID("DOCF", "")
}

func NewSearchProfilePanel() string {
	return newPHID("PANL", "")
}

func NewProject() string {
	return newPHID("PROJ", "")
}

func NewManiphestTransaction() string {
	return newPHID("XACT", "TASK")
}

func NewManiphestTask() string {
	return newPHID("TASK", "")
}

func NewManiphestComment() string {
	return newPHID("XCMT", "")
}

/*
New PHIDs defined by appscode MUST use prefix = ACID to avoid collision with potential future pHabricator applications.
*/

func NewCloudCredential() string {
	return newACID("CRED", "")
}

func NewSSHKey() string {
	return newACID("SSH", "")
}

func NewAuthSSHKey() string {
	return newPHID("AKEY", "")
}

func NewJenkinsAgent() string {
	return newACID("JENT", "")
}

func NewNamespace() string {
	return newACID("NS", "")
}

func NewSecret() string {
	return newACID("SCRT", "")
}

func NewCA() string {
	return newACID("CA", "")
}

func NewCert() string {
	return newACID("CERT", "")
}

func NewKubeCluster() string {
	return newACID("K8S", "C")
}

func NewKubeInstance() string {
	return newACID("K8S", "I")
}

func NewOperation() string {
	return newACID("GOP", "")
}

func NewAlertIncident() string {
	return newACID("ALRT", "INCDT")
}

const (
	PRODUCT_CLUSTER = "CLSR"
	PRODUCT_DB      = "DB"
	PRODUCT_PACKAGE = "PKG"
)

func NewProduct(t string) string {
	switch t {
	case PRODUCT_CLUSTER:
		fallthrough
	case PRODUCT_DB:
		fallthrough
	case PRODUCT_PACKAGE:
		return newACID("PDCT", t)
	default:
		panic("unrecognized product type")
	}
}

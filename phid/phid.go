package phid

import (
	"strings"

	"github.com/appscode/go/crypto/rand"
	"k8s.io/apimachinery/pkg/types"
)

func new(prefix, t, s string) types.UID {
	if s == "" {
		return types.UID(prefix + "-" + t + "-" + rand.Characters(20))
	} else {
		return types.UID(prefix + "-" + t + "-" + s + "-" + rand.Characters(15))
	}
}

func newACID(t, s string) types.UID {
	return new("ACID", t, s)
}

func newPHID(t, s string) types.UID {
	return new("PHID", t, s)
}

func IsPHID(s string) bool {
	// ignores system secret
	// TODO(admin): Should we ignore case sensitivity?
	return strings.HasPrefix(s, "PHID-") || strings.HasPrefix(s, "ACID-")
}

// --- Special Constants ---
func SystemSecret() types.UID {
	return "SYS-SCRT-PHID" // never conflicts with other *IDs
}

// --- Phabricator Defined PHIDs ---
func NewUser() types.UID {
	return newPHID("USER", "")
}

func NewUserPref() types.UID {
	return newPHID("USER", "PREF")
}

func NewUserExternal() types.UID {
	return newPHID("XUSR", "")
}

func NewConfig() types.UID {
	return newPHID("CONF", "")
}

func NewAuthProvider() types.UID {
	return newPHID("AUTH", "")
}

func NewDashboard() types.UID {
	return newPHID("DSHB", "")
}

func NewDashboardTransaction() types.UID {
	return newPHID("XACT", "DSHB")
}

func NewDashboardPanel() types.UID {
	return newPHID("DSHP", "")
}
func NewDashboardPanelTransaction() types.UID {
	return newPHID("XACT", "DSHP")
}

func NewHomeApplication() types.UID {
	return newPHID("APPS", "")
}

func NewProfileMenuItem() types.UID {
	return newPHID("PANL", "")
}

func NewRUri() types.UID {
	return newPHID("RURI", "")
}

func NewCalanderEvent() types.UID {
	return newPHID("CEVT", "")
}

func NewConpherenceThread() types.UID {
	return newPHID("CONP", "")
}

func NewDocumentField() types.UID {
	return newPHID("DOCF", "")
}

func NewSearchProfilePanel() types.UID {
	return newPHID("PANL", "")
}

func NewProject() types.UID {
	return newPHID("PROJ", "")
}

func NewManiphestTransaction() types.UID {
	return newPHID("XACT", "TASK")
}

func NewManiphestTask() types.UID {
	return newPHID("TASK", "")
}

func NewManiphestComment() types.UID {
	return newPHID("XCMT", "")
}

/*
New PHIDs defined by appscode MUST use prefix = ACID to avoid collision with potential future pHabricator applications.
*/

func NewCloudCredential() types.UID {
	return newACID("CRED", "")
}

func NewSSHKey() types.UID {
	return newACID("SSH", "")
}

func NewAuthSSHKey() types.UID {
	return newPHID("AKEY", "")
}

func NewJenkinsAgent() types.UID {
	return newACID("JENT", "")
}

func NewNamespace() types.UID {
	return newACID("NS", "")
}

func NewSecret() types.UID {
	return newACID("SCRT", "")
}

func NewCA() types.UID {
	return newACID("CA", "")
}

func NewCert() types.UID {
	return newACID("CERT", "")
}

func NewKubeCluster() types.UID {
	return newACID("K8S", "C")
}

func NewKubeInstance() types.UID {
	return newACID("K8S", "I")
}

func NewOperation() types.UID {
	return newACID("GOP", "")
}

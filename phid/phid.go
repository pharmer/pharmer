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

/*
New PHIDs defined by appscode MUST use prefix = ACID to avoid collision with potential future pHabricator applications.
*/

func NewCloudCredential() types.UID {
	return newACID("CRED", "")
}

func NewSSHKey() types.UID {
	return newACID("SSH", "")
}

func NewNamespace() types.UID {
	return newACID("NS", "")
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

func NewNodeGroup() types.UID {
	return newACID("K8S", "C")
}

func NewKubeInstance() types.UID {
	return newACID("K8S", "I")
}

func NewOperation() types.UID {
	return newACID("GOP", "")
}

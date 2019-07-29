package v1alpha1

import (
	"bytes"
	//"crypto/sha512"
	//"encoding/base64"
	"sort"
	"strings"
)

const (
	NodeLabelKey_Checksum = "meta.appscode.com/checksum"
)

/*
NodeLabels is used to parse and generate --node-label flag for kubelet.
ref: http://kubernetes.io/docs/admin/kubelet/

NodeLabels also includes functionality to sign and verify appscode.com specific node labels. Verified labels will be
used by cluster mutation engine to update/upgrade nodes.
*/
type NodeLabels map[string]string

func (n NodeLabels) values(appscodeKeysOnly, skipChecksum bool) string {
	keys := make([]string, len(n))
	i := 0
	for k := range n {
		keys[i] = k
		i++
	}
	// sort keys to ensure reproducible checksum calculation
	sort.Strings(keys)

	var buf bytes.Buffer
	i = 0
	for _, k := range keys {
		if appscodeKeysOnly && !strings.Contains(k, ".appscode.com/") ||
			k == NodeLabelKey_Checksum && skipChecksum {
			continue
		}
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(n[k])
		i++
	}
	return buf.String()
}

func (n NodeLabels) String() string {
	return n.values(false, false)
}

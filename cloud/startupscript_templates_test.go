/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloud

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPrekDownloadURL(t *testing.T) {
	testCases := []struct {
		name              string
		kubernetesVersion string
		expectedURL       string
	}{
		{
			name:              "1.14.0",
			kubernetesVersion: "1.14.0",
			expectedURL:       "https://github.com/pharmer/pre-k/releases/download/v1.14.0/pre-k_linux_amd64",
		},
		{
			name:              "1.14.0 with v",
			kubernetesVersion: "v1.14.0",
			expectedURL:       "https://github.com/pharmer/pre-k/releases/download/v1.14.0/pre-k_linux_amd64",
		},
		{
			name:              "1.13.0",
			kubernetesVersion: "1.13.0",
			expectedURL:       "https://cdn.appscode.com/binaries/pre-k/1.13.0/pre-k-linux-amd64",
		},
		{
			name:              "1.13.0 with v",
			kubernetesVersion: "v1.13.0",
			expectedURL:       "https://cdn.appscode.com/binaries/pre-k/1.13.0/pre-k-linux-amd64",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			if err := prekDownloadURL.Execute(buf, TemplateData{
				KubernetesVersion: tc.kubernetesVersion,
			}); err != nil {
				t.Fatal(err)
			}
			if tc.expectedURL != buf.String() {
				t.Errorf("Expected dload url: %v, got %v", tc.expectedURL, buf.String())
			}
		})
	}
}

func TestPrekDownloadScript(t *testing.T) {
	expected := `
curl -fsSL --retry 5 -o pre-k https://github.com/pharmer/pre-k/releases/download/v1.14.0/pre-k_linux_amd64 \
&& chmod +x pre-k \
&& mv pre-k /usr/bin/
`

	buf := bytes.NewBuffer(nil)
	if err := prekDownload.Execute(buf, TemplateData{
		KubernetesVersion: "1.14.0",
	}); err != nil {
		t.Fatal(err)
	}

	if buf.String() != expected {
		t.Errorf("prek download script didn't match, diff (- want, + got)\n%v", cmp.Diff(buf.String(), expected))
	}

	// make sure it doesn't break startup script
	// TODO: add a test for startup script maybe that compares with actual output file?
	if err := StartupScriptTemplate.Execute(buf, TemplateData{
		KubernetesVersion: "1.14.0",
	}); err != nil {
		t.Fatal(err)
	}
}

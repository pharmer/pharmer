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
package aws

import (
	"bytes"
	"html/template"

	"github.com/pkg/errors"
)

const (
	defaultHeader = `#!/usr/bin/env bash

# Copyright 2018 by the contributors
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.

set -o verbose
set -o errexit
set -o nounset
set -o pipefail
`

	bastionBashScript = `{{.Header}}

BASTION_BOOTSTRAP_FILE=bastion_bootstrap.sh
BASTION_BOOTSTRAP=https://s3.amazonaws.com/aws-quickstart/quickstart-linux-bastion/scripts/bastion_bootstrap.sh

curl -s -o $BASTION_BOOTSTRAP_FILE $BASTION_BOOTSTRAP
chmod +x $BASTION_BOOTSTRAP_FILE

# This gets us far enough in the bastion script to be useful.
apt-get -y update && apt-get -y install python-pip
pip install --upgrade pip &> /dev/null

./$BASTION_BOOTSTRAP_FILE --enable true
`
)

type baseUserData struct {
	Header string
}

func generate(kind string, tpl string, data interface{}) (string, error) {
	t, err := template.New(kind).Parse(tpl)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %s template", kind)
	}

	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return "", errors.Wrapf(err, "failed to generate %s template", kind)
	}

	return out.String(), nil
}

// BastionInput defines the context to generate a bastion instance user data.
type BastionInput struct {
	baseUserData
}

// NewBastion returns the user data string to be used on a bastion instance.
func NewBastion(input *BastionInput) (string, error) {
	input.Header = defaultHeader
	return generate("bastion", bastionBashScript, input)
}

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
package kube

import (
	"fmt"
	mrnd "math/rand"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

func GetKubeadmToken() string {
	return fmt.Sprintf("%s.%s", RandStringRunes(6), RandStringRunes(16))
}

func init() {
	mrnd.Seed(time.Now().UnixNano())
}

// Hexadecimal
var letterRunes = []rune("0123456789abcdef")

var (
	// TokenIDRegexpString defines token's id regular expression pattern
	TokenIDRegexpString = "^([a-z0-9]{6})$"
	// TokenIDRegexp is a compiled regular expression of TokenIDRegexpString
	TokenIDRegexp = regexp.MustCompile(TokenIDRegexpString)
	// TokenRegexpString defines id.secret regular expression pattern
	TokenRegexpString = "^([a-z0-9]{6})\\.([a-z0-9]{16})$"
	// TokenRegexp is a compiled regular expression of TokenRegexpString
	TokenRegexp = regexp.MustCompile(TokenRegexpString)
)

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrnd.Intn(len(letterRunes))]
	}
	return string(b)
}

func ParseToken(s string) (string, string, error) {
	split := TokenRegexp.FindStringSubmatch(s)
	if len(split) != 3 {
		return "", "", errors.Errorf("token [%q] was not of form [%q]", s, TokenRegexpString)
	}
	return split[1], split[2], nil
}

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
package credential

type Swift struct {
	CommonSpec
}

func (c Swift) Username() string      { return c.Data[SwiftUsername] }
func (c Swift) Key() string           { return c.Data[SwiftKey] }
func (c Swift) TenantName() string    { return c.Data[SwiftTenantName] }
func (c Swift) TenantAuthURL() string { return c.Data[SwiftTenantAuthURL] }
func (c Swift) Domain() string        { return c.Data[SwiftDomain] }
func (c Swift) Region() string        { return c.Data[SwiftRegion] }
func (c Swift) TenantId() string      { return c.Data[SwiftTenantId] }
func (c Swift) TenantDomain() string  { return c.Data[SwiftTenantDomain] }
func (c Swift) TrustId() string       { return c.Data[SwiftTrustId] }
func (c Swift) StorageURL() string    { return c.Data[SwiftStorageURL] }
func (c Swift) AuthToken() string     { return c.Data[SwiftAuthToken] }

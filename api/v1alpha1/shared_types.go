/*
Copyright 2022 svketen.

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

package v1alpha1

type Config struct {
	Prefix string `json:"prefix,omitempty"`

	Suffix string `json:"suffix,omitempty"`

	Delete bool `json:"delete,omitempty"`

	Connection Connection `json:"connection,omitempty"`
}

type Connection struct {
	Credentials `json:",inline"`

	URL string `json:"url,omitempty"`

	Port int32 `json:"port,omitempty"`
}

type Credentials struct {
	Username string `json:"username,omitempty"`

	PasswordRef string `json:"passwordRef,omitempty"`
}

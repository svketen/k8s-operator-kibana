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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleSpec defines the desired state of Role
type RoleSpec struct {
	Prefix string `json:"prefix,omitempty"`

	Suffix string `json:"suffix,omitempty"`

	Connection Connection `json:"connection,omitempty"`

	Roles []KibanaRole `json:"roles,omitempty"`
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

type KibanaRole struct {
	Name string `json:"name,omitempty"`

	Elasticsearch Elasticsearch `json:"elasticsearch,omitempty"`

	Kibana []Kibana `json:"kibana,omitempty"`
}

type Elasticsearch struct {
	Name string `json:"name,omitempty"`

	ClusterPrivileges []string `json:"cluster,omitempty"`

	RunAsPrivileges []string `json:"run_as,omitempty"`

	IndexPrivileges []IndexPrivileges `json:"indices,omitempty"`
}

type IndexPrivileges struct {
	Names []string `json:"names,omitempty"`

	Privileges []string `json:"privileges,omitempty"`

	AllowRestrictedIndices bool `json:"allow_restricted_indices,omitempty"`
}

type Kibana struct {
	Base    []string      `json:"base,omitempty"`
	Feature KibanaFeature `json:"feature,omitempty"`
	Spaces  []string      `json:"spaces,omitempty"`
}

type KibanaFeature struct {
	AdvancedSettings       []string `json:"advancedSettings,omitempty"`
	Dashboard              []string `json:"dashboard,omitempty"`
	Discover               []string `json:"discover,omitempty"`
	IndexPatterns          []string `json:"indexPatterns,omitempty"`
	SavedObjectsManagement []string `json:"savedObjectsManagement,omitempty"`
	Visualize              []string `json:"visualize,omitempty"`
}

// RoleStatus defines the observed state of Role
type RoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Role is the Schema for the roles API
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleSpec   `json:"spec,omitempty"`
	Status RoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RoleList contains a list of Role
type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Role `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}

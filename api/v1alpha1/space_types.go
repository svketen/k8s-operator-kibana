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

// SpaceSpec defines the desired state of Space
type SpaceSpec struct {
	Config `json:",inline"`

	Spaces []KibanaSpace `json:"spaces,omitempty"`
}

type KibanaSpace struct {
	Id string `json:"id,omitempty"`

	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`

	DisabledFeatures []string `json:"disabledFeatures,omitempty"`
}

// SpaceStatus defines the observed state of Space
type SpaceStatus struct {
	Created int32 `json:"created,omitempty"`
	Updated int32 `json:"updated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Space is the Schema for the spaces API
type Space struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceSpec   `json:"spec,omitempty"`
	Status SpaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SpaceList contains a list of Space
type SpaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Space `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Space{}, &SpaceList{})
}

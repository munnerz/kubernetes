/*
Copyright 2024 The Kubernetes Authors.

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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:prerelease-lifecycle-gen:introduced=1.31

// ScopeDefinition is a definition of a mapping between a scope (name, value) tuple
// and a list of namespace names.
// The metadata.namespace field is used to represent the scope name, and the
// metadata.name field is used to represent the scope value.
// For example, a ScopeDefinition in the namespace 'workspace' with name 'my-workspace'
// would correspond to the scope selector `scope.k8s.io/workspace=my-workspace`.
// A scopes generation field is used to uniquely identify a revision of a scope configuration.
type ScopeDefinition struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec ScopeDefinitionSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type ScopeDefinitionSpec struct {
	// Namespaces is the list of namespaces currently contained within this scope.
	Namespaces []string `json:"namespaces,omitempty" protobuf:"bytes,1,rep,name=namespaces"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:prerelease-lifecycle-gen:introduced=1.31

// ScopeDefinitionList is a collection of ScopeDefinition objects.
type ScopeDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []ScopeDefinition `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}

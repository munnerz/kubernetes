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
// would correspond to the scope selector `workspace=my-workspace`.
// A scopes generation field is used to uniquely identify a revision of a scope configuration.
type ScopeDefinition struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// The name must be of the form `<scope-name>:<scope-value>`, for example: `workspaces:my-workspace`.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the ScopeDefinition.
	Spec ScopeDefinitionSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status of the ScopeDefinition.
	Status ScopeDefinitionStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type ScopeDefinitionSpec struct {
	// Namespaces is a list of static & explicit namespace names to be included in the scope.
	// +listType=set
	Namespaces []string `json:"namespaces,omitempty" protobuf:"bytes,1,rep,name=namespaces"`
}

type ScopeDefinitionStatus struct {
	// ScopeID is a unique identifier for this generation/epoch of mapping.
	ScopeID string `json:"scopeID,omitempty" protobuf:"bytes,1,opt,name=scopeID"`

	// Namespaces is the final set of namespaces that are included within this scope.
	// +listType=set
	Namespaces []string `json:"namespaces,omitempty" protobuf:"bytes,2,rep,name=namespaces"`

	// MinimumResourceVersions are the minimum supported store resource versions for this scope,
	// computed by finding the highest resourceVersion reported from an individual server that it
	// most recently transitioned scopes between.
	// +optional
	// +listType=map
	// +listMapKey=storeID
	MinimumResourceVersions []MinimumResourceVersion `json:"minimumResourceVersions,omitempty" protobuf:"bytes,3,rep,name=minimumResourceVersions"`

	// ServerScopeVersions contains an entry for each (apiServer, store) pair detailing the progress
	// in the store when the last scope ID was applied.
	// +optional
	// +listType=map
	// +listMapKey=apiServerID,storeID
	ServerScopeVersions []ServerScopeVersion `json:"serverScopeVersions,omitempty" protobuf:"bytes,4,rep,name=serverScopeVersions"`
}

// MinimumResourceVersion contains information about a scopes validity within a particular store.
type MinimumResourceVersion struct {
	// The ID of the storage backend in the API server.
	StoreID string `json:"storeID" protobuf:"bytes,1,opt,name=storeID"`

	// ResourceVersion is the minimum supported resource version for this scope in the store.
	ResourceVersion string `json:"resourceVersion" protobuf:"bytes,2,opt,name=resourceVersion"`
}

// ServerScopeVersion contains information on when a particular apiserver first began serving
// requests using a new mapping ID.
// This may not be the FIRST resourceVersion served using this mapping, however it's guaranteed to always
// be AFTER the mapping began to be served.
type ServerScopeVersion struct {
	// The ID of the reporting API server.
	APIServerID string `json:"apiServerID" protobuf:"bytes,1,opt,name=apiServerID"`

	// The ID of the storage backend in the API server.
	// This should be consistent between different servers.
	StoreID string `json:"storeID" protobuf:"bytes,2,opt,name=storeID"`

	// ScopeID is the generation of scope that this apiserver and store is recording progress for.
	ScopeID string `json:"scopeID" protobuf:"bytes,3,opt,name=scopeID"`

	// ResourceVersion is the current resourceVersion of the store at a point at or after this generation
	// of scope began to be used by this server.
	ResourceVersion string `json:"resourceVersion" protobuf:"bytes,4,opt,name=resourceVersion"`
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

//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MinimumResourceVersion) DeepCopyInto(out *MinimumResourceVersion) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MinimumResourceVersion.
func (in *MinimumResourceVersion) DeepCopy() *MinimumResourceVersion {
	if in == nil {
		return nil
	}
	out := new(MinimumResourceVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScopeDefinition) DeepCopyInto(out *ScopeDefinition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScopeDefinition.
func (in *ScopeDefinition) DeepCopy() *ScopeDefinition {
	if in == nil {
		return nil
	}
	out := new(ScopeDefinition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ScopeDefinition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScopeDefinitionList) DeepCopyInto(out *ScopeDefinitionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ScopeDefinition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScopeDefinitionList.
func (in *ScopeDefinitionList) DeepCopy() *ScopeDefinitionList {
	if in == nil {
		return nil
	}
	out := new(ScopeDefinitionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ScopeDefinitionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScopeDefinitionSpec) DeepCopyInto(out *ScopeDefinitionSpec) {
	*out = *in
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScopeDefinitionSpec.
func (in *ScopeDefinitionSpec) DeepCopy() *ScopeDefinitionSpec {
	if in == nil {
		return nil
	}
	out := new(ScopeDefinitionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScopeDefinitionStatus) DeepCopyInto(out *ScopeDefinitionStatus) {
	*out = *in
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.MinimumResourceVersions != nil {
		in, out := &in.MinimumResourceVersions, &out.MinimumResourceVersions
		*out = make([]MinimumResourceVersion, len(*in))
		copy(*out, *in)
	}
	if in.ServerScopeVersions != nil {
		in, out := &in.ServerScopeVersions, &out.ServerScopeVersions
		*out = make([]ServerScopeVersion, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScopeDefinitionStatus.
func (in *ScopeDefinitionStatus) DeepCopy() *ScopeDefinitionStatus {
	if in == nil {
		return nil
	}
	out := new(ScopeDefinitionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServerScopeVersion) DeepCopyInto(out *ServerScopeVersion) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServerScopeVersion.
func (in *ServerScopeVersion) DeepCopy() *ServerScopeVersion {
	if in == nil {
		return nil
	}
	out := new(ServerScopeVersion)
	in.DeepCopyInto(out)
	return out
}

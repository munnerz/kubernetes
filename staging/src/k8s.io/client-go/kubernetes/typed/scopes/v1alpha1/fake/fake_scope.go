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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1alpha1 "k8s.io/api/scopes/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	scopesv1alpha1 "k8s.io/client-go/applyconfigurations/scopes/v1alpha1"
	testing "k8s.io/client-go/testing"
)

// FakeScopes implements ScopeInterface
type FakeScopes struct {
	Fake *FakeScopesV1alpha1
}

var scopesResource = v1alpha1.SchemeGroupVersion.WithResource("scopes")

var scopesKind = v1alpha1.SchemeGroupVersion.WithKind("Scope")

// Get takes name of the scope, and returns the corresponding scope object, and an error if there is any.
func (c *FakeScopes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Scope, err error) {
	emptyResult := &v1alpha1.Scope{}
	obj, err := c.Fake.
		Invokes(testing.NewRootGetActionWithOptions(scopesResource, name, options), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Scope), err
}

// List takes label and field selectors, and returns the list of Scopes that match those selectors.
func (c *FakeScopes) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ScopeList, err error) {
	emptyResult := &v1alpha1.ScopeList{}
	obj, err := c.Fake.
		Invokes(testing.NewRootListActionWithOptions(scopesResource, scopesKind, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ScopeList{ListMeta: obj.(*v1alpha1.ScopeList).ListMeta}
	for _, item := range obj.(*v1alpha1.ScopeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested scopes.
func (c *FakeScopes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchActionWithOptions(scopesResource, opts))
}

// Create takes the representation of a scope and creates it.  Returns the server's representation of the scope, and an error, if there is any.
func (c *FakeScopes) Create(ctx context.Context, scope *v1alpha1.Scope, opts v1.CreateOptions) (result *v1alpha1.Scope, err error) {
	emptyResult := &v1alpha1.Scope{}
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateActionWithOptions(scopesResource, scope, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Scope), err
}

// Update takes the representation of a scope and updates it. Returns the server's representation of the scope, and an error, if there is any.
func (c *FakeScopes) Update(ctx context.Context, scope *v1alpha1.Scope, opts v1.UpdateOptions) (result *v1alpha1.Scope, err error) {
	emptyResult := &v1alpha1.Scope{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateActionWithOptions(scopesResource, scope, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Scope), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeScopes) UpdateStatus(ctx context.Context, scope *v1alpha1.Scope, opts v1.UpdateOptions) (result *v1alpha1.Scope, err error) {
	emptyResult := &v1alpha1.Scope{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceActionWithOptions(scopesResource, "status", scope, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Scope), err
}

// Delete takes name of the scope and deletes it. Returns an error if one occurs.
func (c *FakeScopes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(scopesResource, name, opts), &v1alpha1.Scope{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeScopes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionActionWithOptions(scopesResource, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ScopeList{})
	return err
}

// Patch applies the patch and returns the patched scope.
func (c *FakeScopes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Scope, err error) {
	emptyResult := &v1alpha1.Scope{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(scopesResource, name, pt, data, opts, subresources...), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Scope), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied scope.
func (c *FakeScopes) Apply(ctx context.Context, scope *scopesv1alpha1.ScopeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Scope, err error) {
	if scope == nil {
		return nil, fmt.Errorf("scope provided to Apply must not be nil")
	}
	data, err := json.Marshal(scope)
	if err != nil {
		return nil, err
	}
	name := scope.Name
	if name == nil {
		return nil, fmt.Errorf("scope.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.Scope{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(scopesResource, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Scope), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeScopes) ApplyStatus(ctx context.Context, scope *scopesv1alpha1.ScopeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Scope, err error) {
	if scope == nil {
		return nil, fmt.Errorf("scope provided to Apply must not be nil")
	}
	data, err := json.Marshal(scope)
	if err != nil {
		return nil, err
	}
	name := scope.Name
	if name == nil {
		return nil, fmt.Errorf("scope.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.Scope{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(scopesResource, *name, types.ApplyPatchType, data, opts.ToPatchOptions(), "status"), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Scope), err
}

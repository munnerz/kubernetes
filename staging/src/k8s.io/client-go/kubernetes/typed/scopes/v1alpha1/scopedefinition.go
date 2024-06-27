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

package v1alpha1

import (
	"context"

	v1alpha1 "k8s.io/api/scopes/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	scopesv1alpha1 "k8s.io/client-go/applyconfigurations/scopes/v1alpha1"
	gentype "k8s.io/client-go/gentype"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

// ScopeDefinitionsGetter has a method to return a ScopeDefinitionInterface.
// A group's client should implement this interface.
type ScopeDefinitionsGetter interface {
	ScopeDefinitions() ScopeDefinitionInterface
}

// ScopeDefinitionInterface has methods to work with ScopeDefinition resources.
type ScopeDefinitionInterface interface {
	Create(ctx context.Context, scopeDefinition *v1alpha1.ScopeDefinition, opts v1.CreateOptions) (*v1alpha1.ScopeDefinition, error)
	Update(ctx context.Context, scopeDefinition *v1alpha1.ScopeDefinition, opts v1.UpdateOptions) (*v1alpha1.ScopeDefinition, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, scopeDefinition *v1alpha1.ScopeDefinition, opts v1.UpdateOptions) (*v1alpha1.ScopeDefinition, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ScopeDefinition, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ScopeDefinitionList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ScopeDefinition, err error)
	Apply(ctx context.Context, scopeDefinition *scopesv1alpha1.ScopeDefinitionApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ScopeDefinition, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, scopeDefinition *scopesv1alpha1.ScopeDefinitionApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ScopeDefinition, err error)
	ScopeDefinitionExpansion
}

// scopeDefinitions implements ScopeDefinitionInterface
type scopeDefinitions struct {
	*gentype.ClientWithListAndApply[*v1alpha1.ScopeDefinition, *v1alpha1.ScopeDefinitionList, *scopesv1alpha1.ScopeDefinitionApplyConfiguration]
}

// newScopeDefinitions returns a ScopeDefinitions
func newScopeDefinitions(c *ScopesV1alpha1Client) *scopeDefinitions {
	return &scopeDefinitions{
		gentype.NewClientWithListAndApply[*v1alpha1.ScopeDefinition, *v1alpha1.ScopeDefinitionList, *scopesv1alpha1.ScopeDefinitionApplyConfiguration](
			"scopedefinitions",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v1alpha1.ScopeDefinition { return &v1alpha1.ScopeDefinition{} },
			func() *v1alpha1.ScopeDefinitionList { return &v1alpha1.ScopeDefinitionList{} }),
	}
}

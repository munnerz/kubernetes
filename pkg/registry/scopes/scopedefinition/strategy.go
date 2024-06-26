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

package scopedefinition

import (
	"context"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/apis/scopes"
	"k8s.io/kubernetes/pkg/apis/scopes/validation"
)

// scopeDefinitionStrategy implements verification logic for daemon sets.
type scopeDefinitionStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating ScopeDefinition objects.
var Strategy = scopeDefinitionStrategy{legacyscheme.Scheme, names.SimpleNameGenerator}

// NamespaceScoped returns false because all ScopeDefinitions need to be cluster scoped
func (scopeDefinitionStrategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (scopeDefinitionStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{}

	return fields
}

// PrepareForCreate does not do anything.
func (scopeDefinitionStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update (currently none)
func (scopeDefinitionStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {}

// Validate validates a new daemon set.
func (scopeDefinitionStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	scopeDefinition := obj.(*scopes.ScopeDefinition)
	return validation.ValidateScopeDefinition(scopeDefinition)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (scopeDefinitionStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	var warnings []string
	return warnings
}

// Canonicalize normalizes the object after validation.
func (scopeDefinitionStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is currently not permitted, though it could be in future.
func (scopeDefinitionStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (scopeDefinitionStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newScopeDefinition := obj.(*scopes.ScopeDefinition)
	oldScopeDefinition := old.(*scopes.ScopeDefinition)

	allErrs := validation.ValidateScopeDefinition(obj.(*scopes.ScopeDefinition))
	allErrs = append(allErrs, validation.ValidateScopeDefinitionUpdate(newScopeDefinition, oldScopeDefinition)...)

	return allErrs
}

// WarningsOnUpdate returns warnings for the given update.
func (scopeDefinitionStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	return warnings
}

// AllowUnconditionalUpdate is the default update policy for scope definition objects.
// todo: work out if this is correct
func (scopeDefinitionStrategy) AllowUnconditionalUpdate() bool {
	return true
}

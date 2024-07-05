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

package scope

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

// scopeStrategy implements verification logic for Scope objects
type scopeStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Scope objects.
var Strategy = scopeStrategy{legacyscheme.Scheme, names.SimpleNameGenerator}

// NamespaceScoped returns false because all Scopes need to be cluster scoped
func (scopeStrategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (scopeStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"scopes.k8s.io/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}
}

// PrepareForCreate ensures status is not set.
func (scopeStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	scope := obj.(*scopes.Scope)
	scope.Status = scopes.ScopeStatus{}
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (scopeStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newScope := obj.(*scopes.Scope)
	oldScope := old.(*scopes.Scope)

	// update is not allowed to set status
	newScope.Status = oldScope.Status
}

// Validate validates a new daemon set.
func (scopeStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	scope := obj.(*scopes.Scope)
	return validation.ValidateScope(scope)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (scopeStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	var warnings []string
	return warnings
}

// Canonicalize normalizes the object after validation.
func (scopeStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is currently not permitted, though it could be in future.
func (scopeStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (scopeStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newScope := obj.(*scopes.Scope)
	oldScope := old.(*scopes.Scope)

	allErrs := validation.ValidateScope(obj.(*scopes.Scope))
	allErrs = append(allErrs, validation.ValidateScopeUpdate(newScope, oldScope)...)

	return allErrs
}

// WarningsOnUpdate returns warnings for the given update.
func (scopeStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	return warnings
}

// AllowUnconditionalUpdate is the default update policy for scope definition objects.
// todo: work out if this is correct
func (scopeStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type scopeStatusStrategy struct {
	scopeStrategy
}

// StatusStrategy is the default logic invoked when updating object status.
var StatusStrategy = scopeStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (scopeStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"scopes.k8s.io/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (scopeStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newScope := obj.(*scopes.Scope)
	oldScope := old.(*scopes.Scope)
	newScope.Spec = oldScope.Spec
}

func (scopeStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateScopeStatusUpdate(obj.(*scopes.Scope), old.(*scopes.Scope))
}

// WarningsOnUpdate returns warnings for the given update.
func (scopeStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

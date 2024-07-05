/*
Copyright 2020 The Kubernetes Authors.

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

package validation

import (
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/validation/field"
	apivalidation "k8s.io/kubernetes/pkg/apis/core/validation"
	"k8s.io/kubernetes/pkg/apis/scopes"
)

// ValidateScope validate the storage version object.
func ValidateScope(sv *scopes.Scope) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&sv.ObjectMeta, false, ValidateScopeName, field.NewPath("metadata"))...)
	allErrs = append(allErrs, validateScopeSpec(sv.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateScopeName is a ValidateNameFunc for storage version names
func ValidateScopeName(name string, prefix bool) []string {
	// todo: verify it is a valid string in a label key (when prefixed with 'scope.k8s.io/')
	var allErrs []string
	return allErrs
}

// ValidateScopeUpdate tests if an update to a Scope is valid.
func ValidateScopeUpdate(sv, oldSV *scopes.Scope) field.ErrorList {
	// no error since ScopeSpec is an empty spec
	return field.ErrorList{}
}

// ValidateScopeSpecUpdate tests if an update to a ScopeSpec is valid.
func ValidateScopeSpecUpdate(sv, oldSV *scopes.Scope) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, validateScopeSpec(sv.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateScopeSpec(ss scopes.ScopeSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	// todo: verify namespace names in the list
	return allErrs
}

// ValidateScopeStatusUpdate tests if an update to a ScopeStatus is valid.
func ValidateScopeStatusUpdate(sv, oldSV *scopes.Scope) field.ErrorList {
	var allErrs field.ErrorList
	fldPath := field.NewPath("status")
	if !apiequality.Semantic.DeepEqual(sv.Status.Namespaces, oldSV.Status.Namespaces) {
		if sv.Status.ScopeID == oldSV.Status.ScopeID {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("scopeID"), "status.scopeID must be updated when status.namespaces is changed"))
		}
	}
	// todo: verify all modified conditions are setting the scopeID to status.scopeID.
	return allErrs
}

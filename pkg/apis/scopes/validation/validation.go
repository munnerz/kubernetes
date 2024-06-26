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
	"k8s.io/apimachinery/pkg/util/validation/field"
	apivalidation "k8s.io/kubernetes/pkg/apis/core/validation"
	"k8s.io/kubernetes/pkg/apis/scopes"
)

// ValidateScopeDefinition validate the storage version object.
func ValidateScopeDefinition(sv *scopes.ScopeDefinition) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&sv.ObjectMeta, true, ValidateScopeDefinitionName, field.NewPath("metadata"))...)
	allErrs = append(allErrs, validateScopeDefinitionSpec(sv.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateScopeDefinitionName is a ValidateNameFunc for storage version names
func ValidateScopeDefinitionName(name string, prefix bool) []string {
	// todo: verify it is a valid string in a label key (when prefixed with 'scope.k8s.io/')
	var allErrs []string
	return allErrs
}

// ValidateScopeDefinitionUpdate tests if an update to a ScopeDefinition is valid.
func ValidateScopeDefinitionUpdate(sv, oldSV *scopes.ScopeDefinition) field.ErrorList {
	// no error since ScopeDefinitionSpec is an empty spec
	return field.ErrorList{}
}

// ValidateScopeDefinitionSpecUpdate tests if an update to a ScopeDefinitionSpec is valid.
func ValidateScopeDefinitionSpecUpdate(sv, oldSV *scopes.ScopeDefinition) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, validateScopeDefinitionSpec(sv.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateScopeDefinitionSpec(ss scopes.ScopeDefinitionSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	// todo: verify namespace names in the list
	return allErrs
}

/*
Copyright 2016 The Kubernetes Authors.

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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/certificates"
	apivalidation "k8s.io/kubernetes/pkg/apis/core/validation"
)

// validateCSR validates the signature and formatting of a base64-wrapped,
// PEM-encoded PKCS#10 certificate signing request. If this is invalid, we must
// not accept the CSR for further processing.
func validateCSR(obj *certificates.CertificateSigningRequest) error {
	csr, err := certificates.ParseCSR(obj)
	if err != nil {
		return err
	}
	// check that the signature is valid
	err = csr.CheckSignature()
	if err != nil {
		return err
	}
	return nil
}

// We don't care what you call your certificate requests.
func ValidateCertificateRequestName(name string, prefix bool) []string {
	return nil
}

func ValidateCertificateSigningRequest(csr *certificates.CertificateSigningRequest) field.ErrorList {
	isNamespaced := false
	allErrs := apivalidation.ValidateObjectMeta(&csr.ObjectMeta, isNamespaced, ValidateCertificateRequestName, field.NewPath("metadata"))
	err := validateCSR(csr)

	specPath := field.NewPath("spec")

	if err != nil {
		allErrs = append(allErrs, field.Invalid(specPath.Child("request"), csr.Spec.Request, fmt.Sprintf("%v", err)))
	}
	if len(csr.Spec.Usages) == 0 {
		allErrs = append(allErrs, field.Required(specPath.Child("usages"), "usages must be provided"))
	}
	if csr.Spec.SignerName == nil {
		allErrs = append(allErrs, field.Required(specPath.Child("signerName"), "signerName must be provided"))
	} else {
		// ensure signerName is of the form domain.com/something and up to 571 characters
		if len(*csr.Spec.SignerName) > 571 {
			allErrs = append(allErrs, field.TooLong(specPath.Child("signerName"), *csr.Spec.SignerName, 571))
		}
		parts := strings.Split(*csr.Spec.SignerName, "/")
		if len(parts) != 2 {
			allErrs = append(allErrs, field.Invalid(specPath.Child("signerName"), *csr.Spec.SignerName, "must be a fully qualified domain of the form 'example.com/signer-name'"))
		} else {
			// check the domain portion to ensure it is a valid domain label
			if errs := validation.NameIsDNSSubdomain(parts[0], false); errs != nil {
				for _, err := range errs {
					allErrs = append(allErrs, field.Invalid(specPath.Child("signerName"), *csr.Spec.SignerName, err))
				}
			}

			// TODO: should we also validate the 'path' component?
		}
	}
	return allErrs
}

func ValidateCertificateSigningRequestUpdate(newCSR, oldCSR *certificates.CertificateSigningRequest) field.ErrorList {
	validationErrorList := ValidateCertificateSigningRequest(newCSR)
	metaUpdateErrorList := apivalidation.ValidateObjectMetaUpdate(&newCSR.ObjectMeta, &oldCSR.ObjectMeta, field.NewPath("metadata"))
	return append(validationErrorList, metaUpdateErrorList...)
}

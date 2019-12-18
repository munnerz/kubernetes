/*
Copyright 2017 The Kubernetes Authors.

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

package v1beta1

import (
	"crypto/x509"
	"reflect"
	"strings"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_CertificateSigningRequestSpec(obj *certificatesv1beta1.CertificateSigningRequestSpec) {
	if obj.Usages == nil {
		obj.Usages = []certificatesv1beta1.KeyUsage{certificatesv1beta1.UsageDigitalSignature, certificatesv1beta1.UsageKeyEncipherment}
	}

	if obj.SignerName == "" {
		// default to legacy-unknown if anything else fails
		obj.SignerName = certificatesv1beta1.LegacyUnknownSignerName
		csr, err := ParseCSR(&certificatesv1beta1.CertificateSigningRequest{Spec: *obj})
		if err != nil {
			klog.Errorf("failed to parse CSR: %v", err)
			return
		}

		if isKubeletClientCSR(csr, obj.Usages) {
			obj.SignerName = certificatesv1beta1.KubeletClientSignerName
		}
	}
}

func isKubeletClientCSR(req *x509.CertificateRequest, usages []certificatesv1beta1.KeyUsage) bool {
	if !reflect.DeepEqual([]string{"system:nodes"}, req.Subject.Organization) {
		return false
	}

	if !strings.HasPrefix(req.Subject.CommonName, "system:node:") {
		return false
	}

	requiredUsages := []certificatesv1beta1.KeyUsage{
		certificatesv1beta1.UsageDigitalSignature,
		certificatesv1beta1.UsageKeyEncipherment,
		certificatesv1beta1.UsageClientAuth,
	}
	if !equalUnsorted(requiredUsages, usages) {
		return false
	}

	return true
}

// equalUnsorted compares two []string for equality of contents regardless of
// the order of the elements
func equalUnsorted(left, right []certificatesv1beta1.KeyUsage) bool {
	l := sets.NewString()
	for _, s := range left {
		l.Insert(string(s))
	}
	r := sets.NewString()
	for _, s := range right {
		r.Insert(string(s))
	}
	return l.Equal(r)
}

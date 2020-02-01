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

package subjectrestrictions

import (
	"context"
	"fmt"
	"io"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/klog"
	certificatesapi "k8s.io/kubernetes/pkg/apis/certificates"
)

const PluginName = "CertificateSubjectRestrictions"

func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return newPlugin(), nil
	})
}

type Plugin struct {
	*admission.Handler
}

func (p *Plugin) ValidateInitialization() error {
	return nil
}

var _ admission.ValidationInterface = &Plugin{}

func newPlugin() *Plugin {
	return &Plugin{
		Handler: admission.NewHandler(admission.Create),
	}
}

func (p *Plugin) Validate(_ context.Context, a admission.Attributes, _ admission.ObjectInterfaces) error {
	if a.GetResource().GroupResource() != certificatesapi.Resource("certificatesigningrequests") || a.GetSubresource() != "" {
		return nil
	}

	csr, ok := a.GetObject().(*certificatesapi.CertificateSigningRequest)
	if !ok {
		return admission.NewForbidden(a, fmt.Errorf("expected type CertificateSigningRequest, got: %T", a.GetObject()))
	}

	if *csr.Spec.SignerName != certificatesv1beta1.KubeAPIServerClientSignerName {
		return nil
	}

	csrParsed, err := certificatesapi.ParseCSR(csr)
	if err != nil {
		return admission.NewForbidden(a, fmt.Errorf("failed to parse CSR: %v", err))
	}

	for _, group := range csrParsed.Subject.Organization {
		if group == "system:masters" {
			klog.V(4).Infof("CSR %s rejected by admission plugin %s for attempting to use signer %s with system:masters group",
				csr.Name, PluginName, certificatesv1beta1.KubeAPIServerClientSignerName)
			return admission.NewForbidden(a, fmt.Errorf("use of %s signer with system:masters group is not allowed",
				certificatesv1beta1.KubeAPIServerClientSignerName))
		}
	}

	return nil
}

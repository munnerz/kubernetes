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

package certificates

import (
	"context"
	"fmt"
	"io"

	"k8s.io/klog"

	"k8s.io/apiserver/pkg/admission"
	genericadmissioninit "k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	api "k8s.io/kubernetes/pkg/apis/certificates"
)

// PluginName is a string with the name of the plugin
const PluginName = "CertificateApproval"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return newPlugin(), nil
	})
}

// Plugin holds state for and implements the admission plugin.
type Plugin struct {
	*admission.Handler
	authz            authorizer.Authorizer
}

// SetAuthorizer sets the authorizer.
func (p *Plugin) SetAuthorizer(authz authorizer.Authorizer) {
	p.authz = authz
}

// ValidateInitialization ensures an authorizer is set.
func (p *Plugin) ValidateInitialization() error {
	if p.authz == nil {
		return fmt.Errorf("%s requires an authorizer", PluginName)
	}
	return nil
}

var _ admission.ValidationInterface = &Plugin{}
var _ genericadmissioninit.WantsAuthorizer = &Plugin{}

// newPlugin creates a new PSP admission plugin.
func newPlugin() *Plugin {
	return &Plugin{
		Handler:          admission.NewHandler(admission.Create),
	}
}

// Validate verifies attributes against the PodSecurityPolicy
func (p *Plugin) Validate(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	csr := a.GetObject().(*api.CertificateSigningRequest)

	if isAuthorizedForPolicy(ctx, a.GetUserInfo(), csr.Spec.SignerName, csr.Name, p.authz) {
		return nil
	}

	// we didn't validate against any provider, reject the pod and give the errors for each attempt
	klog.V(4).Infof("user not permitted to approve CertificateSigningRequest %q with signerName %q: %v", csr.Name, csr.Spec.SignerName)
	return admission.NewForbidden(a, fmt.Errorf("user not permitted to approve requests with signerName %q"))
}

// isAuthorizedForPolicy returns true if info is authorized to perform the "approve/signerName" verb on the CSR resource.
func isAuthorizedForPolicy(ctx context.Context, info user.Info, signerName, csrName string, authz authorizer.Authorizer) bool {
	if info == nil {
		return false
	}
	attr := buildAttributes(info, signerName, csrName)
	decision, reason, err := authz.Authorize(ctx, attr)
	if err != nil {
		klog.V(5).Infof("cannot authorize for policy: %v,%v", reason, err)
	}
	return (decision == authorizer.DecisionAllow)
}

// buildAttributes builds an attributes record for a SAR based on the user info and policy.
func buildAttributes(info user.Info, signerName, csrName string) authorizer.Attributes {
	// check against the namespace that the pod is being created in to allow per-namespace PSP grants.
	attr := authorizer.AttributesRecord{
		User:            info,
		Verb:            "create",
		Name:            csrName,
		APIGroup:        "certificates.k8s.io",
		APIVersion:      "*",
		Resource:        "certificatesigningrequests/approve/" + signerName,
		ResourceRequest: true,
	}
	return attr
}

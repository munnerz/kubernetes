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
	authz authorizer.Authorizer
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

// newPlugin creates a new CSR approval admission plugin
func newPlugin() *Plugin {
	return &Plugin{
		Handler: admission.NewHandler(admission.Update),
	}
}

// Validate verifies that the requesting user has permission to approve
// CertificateSigningRequests for the specified signerName.
func (p *Plugin) Validate(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	// Ignore all calls to anything other than 'certificatesigningrequests/approval'.
	// Ignore all operations other than UPDATE.
	if a.GetSubresource() != "approval" ||
		a.GetResource().GroupResource() != api.Resource("certificatesigningrequests") ||
		a.GetOperation() != admission.Update {
		return nil
	}

	csr := a.GetObject().(*api.CertificateSigningRequest)

	// if signerName is not set, the approving user must have permission to
	// approve *all* signerNames (i.e. no 'resourceNames' attribute provided).
	// TODO: should we apply defaulting logic here and test for legacy-unknown,
	//  or alternatively always allow requests if the signerName is empty as it
	//  implies we are speaking to a non-signerName aware client
	signerName := ""
	if csr.Spec.SignerName != nil {
		signerName = *csr.Spec.SignerName
	}

	if isAuthorizedForPolicy(ctx, a.GetUserInfo(), signerName, csr.Name, p.authz) {
		return nil
	}

	// we didn't validate against any provider, reject the pod and give the errors for each attempt
	klog.V(4).Infof("user not permitted to approve CertificateSigningRequest %q with signerName %q", csr.Name, signerName)
	return admission.NewForbidden(a, fmt.Errorf("user not permitted to approve requests with signerName %q", signerName))
}

// isAuthorizedForPolicy returns true if info is authorized to perform the "approve" verb on the synthetic
// 'certificatesigners' resource.
func isAuthorizedForPolicy(ctx context.Context, info user.Info, signerName, csrName string, authz authorizer.Authorizer) bool {
	if info == nil {
		return false
	}
	attr := buildAttributes(info, signerName)
	decision, reason, err := authz.Authorize(ctx, attr)
	if err != nil {
		klog.V(5).Infof("cannot authorize for policy: %v,%v", reason, err)
	}
	return (decision == authorizer.DecisionAllow)
}

// buildAttributes builds an attributes record for a SAR based on the user info and policy.
func buildAttributes(info user.Info, signerName string) authorizer.Attributes {
	attr := authorizer.AttributesRecord{
		User:            info,
		Verb:            "approve",
		Name:            signerName,
		APIGroup:        "certificates.k8s.io",
		APIVersion:      "*",
		Resource:        "certificatesigners",
		ResourceRequest: true,
	}
	return attr
}

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

package signing

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"

	"k8s.io/klog"

	"k8s.io/apiserver/pkg/admission"
	genericadmissioninit "k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	api "k8s.io/kubernetes/pkg/apis/certificates"
)

// PluginName is a string with the name of the plugin
const PluginName = "CertificateSigning"

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
	if a.GetSubresource() != "status" ||
		a.GetResource().GroupResource() != api.Resource("certificatesigningrequests") ||
		a.GetOperation() != admission.Update {
		return nil
	}

	oldCSR := a.GetOldObject().(*api.CertificateSigningRequest)
	csr := a.GetObject().(*api.CertificateSigningRequest)

	// only run if the status.certificate field has been changed
	if reflect.DeepEqual(oldCSR.Status.Certificate, csr.Status.Certificate) {
		return nil
	}

	// It is safe for us to dereference signerName because the defaulting logic will run before this admission
	// controller is executed, meaning we know it is set even if an older client that is not aware of the signerName
	// field is submitting the signed certificate.
	signerName := *csr.Spec.SignerName

	if isAuthorizedForPolicy(ctx, a.GetUserInfo(), signerName, p.authz) {
		return nil
	}

	// we didn't validate against any provider, reject the pod and give the errors for each attempt
	klog.V(4).Infof("user not permitted to sign CertificateSigningRequest %q with signerName %q", csr.Name, signerName)
	return admission.NewForbidden(a, fmt.Errorf("user not permitted to sign requests with signerName %q", signerName))
}

// isAuthorizedForPolicy returns true if info is authorized to perform the "sign" verb on the synthetic
// 'signers' resource with the given signerName.
// It will also check if the user has permission to perform the "sign" verb against the domain portion of
// the signerName + /*, i.e. `kubernetes.io/*` iff the user doesn't have explicit permission on the named signer.
func isAuthorizedForPolicy(ctx context.Context, info user.Info, signerName string, authz authorizer.Authorizer) bool {
	if info == nil {
		return false
	}

	// First check if the user has explicit permission to approve for the given signerName.
	attr := buildAttributes(info, signerName)
	decision, reason, err := authz.Authorize(ctx, attr)
	if err != nil {
		klog.V(5).Infof("cannot authorize for policy: %v,%v", reason, err)
		return false
	}
	if decision == authorizer.DecisionAllow {
		return true
	}

	// If not, check if the user has wildcard permissions to approve for the domain portion of the signerName, e.g.
	// 'kubernetes.io/*'.
	attr = buildWildcardAttributes(info, signerName)
	decision, reason, err = authz.Authorize(ctx, attr)
	if err != nil {
		klog.V(5).Infof("cannot authorize for policy: %v,%v", reason, err)
		return false
	}
	if decision == authorizer.DecisionAllow {
		return true
	}

	return false
}

// buildAttributes builds an attributes record for a SAR based on the user info and policy.
func buildAttributes(info user.Info, signerName string) authorizer.Attributes {
	return buildAttributeRecord(info, signerName)
}

func buildWildcardAttributes(info user.Info, signerName string) authorizer.Attributes {
	parts := strings.Split(signerName, "/")
	domain := parts[0]
	return buildAttributeRecord(info, domain+"/*")
}

func buildAttributeRecord(info user.Info, name string) authorizer.Attributes {
	return authorizer.AttributesRecord{
		User:            info,
		Verb:            "sign",
		Name:            name,
		APIGroup:        "certificates.k8s.io",
		APIVersion:      "*",
		Resource:        "signers",
		ResourceRequest: true,
	}
}

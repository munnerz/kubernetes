package subjectrestrictions

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"testing"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	certificatesapi "k8s.io/kubernetes/pkg/apis/certificates"
)

func TestPlugin_Validate(t *testing.T) {
	tests := []struct {
		name    string
		a       admission.Attributes
		wantErr string
	}{
		{
			name: "ignored resource",
			a: &testAttributes{
				resource: schema.GroupResource{
					Group:    "foo",
					Resource: "bar",
				},
			},
			wantErr: "",
		},
		{
			name: "ignored subresource",
			a: &testAttributes{
				resource:    certificatesapi.Resource("certificatesigningrequests"),
				subresource: "approve",
			},
			wantErr: "",
		},
		{
			name: "wrong type",
			a: &testAttributes{
				resource: certificatesapi.Resource("certificatesigningrequests"),
				obj:      &certificatesapi.CertificateSigningRequestList{},
				name:     "panda",
			},
			wantErr: `certificatesigningrequests.certificates.k8s.io "panda" is forbidden: expected type CertificateSigningRequest, got: *certificates.CertificateSigningRequestList`,
		},
		{
			name: "some other signer",
			a: &testAttributes{
				resource: certificatesapi.Resource("certificatesigningrequests"),
				obj: &certificatesapi.CertificateSigningRequest{Spec: certificatesapi.CertificateSigningRequestSpec{
					Request:    pemWithGroup("system:masters"),
					SignerName: func(s string) *string { return &s }(certificatesv1beta1.KubeAPIServerClientKubeletSignerName),
				}},
			},
			wantErr: "",
		},
		{
			name: "invalid request",
			a: &testAttributes{
				resource: certificatesapi.Resource("certificatesigningrequests"),
				obj: &certificatesapi.CertificateSigningRequest{Spec: certificatesapi.CertificateSigningRequestSpec{
					Request:    []byte("this is not a CSR"),
					SignerName: func(s string) *string { return &s }(certificatesv1beta1.KubeAPIServerClientSignerName),
				}},
				name: "bear",
			},
			wantErr: `certificatesigningrequests.certificates.k8s.io "bear" is forbidden: failed to parse CSR: PEM block type must be CERTIFICATE REQUEST`,
		},
		{
			name: "some other group",
			a: &testAttributes{
				resource: certificatesapi.Resource("certificatesigningrequests"),
				obj: &certificatesapi.CertificateSigningRequest{Spec: certificatesapi.CertificateSigningRequestSpec{
					Request:    pemWithGroup("system:admin"),
					SignerName: func(s string) *string { return &s }(certificatesv1beta1.KubeAPIServerClientSignerName),
				}},
			},
			wantErr: "",
		},
		{
			name: "request for system:masters",
			a: &testAttributes{
				resource: certificatesapi.Resource("certificatesigningrequests"),
				obj: &certificatesapi.CertificateSigningRequest{Spec: certificatesapi.CertificateSigningRequestSpec{
					Request:    pemWithGroup("system:masters"),
					SignerName: func(s string) *string { return &s }(certificatesv1beta1.KubeAPIServerClientSignerName),
				}},
				name: "pooh",
			},
			wantErr: `certificatesigningrequests.certificates.k8s.io "pooh" is forbidden: use of kubernetes.io/kube-apiserver-client signer with system:masters group is not allowed`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{}
			if err := p.Validate(nil, tt.a, nil); errStr(err) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testAttributes struct {
	resource    schema.GroupResource
	subresource string
	obj         runtime.Object
	name        string

	admission.Attributes // nil panic if any other methods called
}

func (t *testAttributes) GetResource() schema.GroupVersionResource {
	return t.resource.WithVersion("ignored")
}

func (t *testAttributes) GetSubresource() string {
	return t.subresource
}

func (t *testAttributes) GetObject() runtime.Object {
	return t.obj
}

func (t *testAttributes) GetName() string {
	return t.name
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	es := err.Error()
	if len(es) == 0 {
		panic("invalid empty error")
	}
	return es
}

func pemWithGroup(group string) []byte {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{group},
		},
	}

	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		panic(err)
	}

	csrPemBlock := &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	}

	p := pem.EncodeToMemory(csrPemBlock)
	if p == nil {
		panic("invalid pem block")
	}

	return p
}

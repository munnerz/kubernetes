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

package certificates

import (
	"testing"

	certv1beta1 "k8s.io/api/certificates/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"k8s.io/kubernetes/test/integration/framework"
)

// Verifies that the CSR approval admission plugin correctly enforces that a
// user has permission to approve CSRs for the named signer
func TestCSRSignerNameApprovalPlugin(t *testing.T) {
	_, s, closeFn := framework.RunAMaster(nil)
	defer closeFn()

	client := clientset.NewForConfigOrDie(&restclient.Config{Host: s.URL, ContentConfig: restclient.ContentConfig{GroupVersion: &schema.GroupVersion{Group: "", Version: "v1"}}})

	csr1 := buildTestingCSR("csr-1", "signer-1")
	csr2 := buildTestingCSR("csr-2", "signer-2")

	testUsername := "testuser"
	// grant 'testuser' permission to approve CSRs for the signerName 'signer-1' ONLY.
	cleanupFn, err := grantUserPermissionToSignFor(client, testUsername, "signer-1")
	if err != nil {
		t.Errorf("unable to create test fixture RBAC rules: %v", err)
		return
	}
	defer cleanupFn()

	// create test fixtures
	if csr1, err = client.CertificatesV1beta1().CertificateSigningRequests().Create(csr1); err != nil {
		t.Errorf("unable to create test csr: %v", err)
		return
	}
	defer client.CertificatesV1beta1().CertificateSigningRequests().Delete(csr1.Name, nil)
	if csr2, err = client.CertificatesV1beta1().CertificateSigningRequests().Create(csr2); err != nil {
		t.Errorf("unable to create test csr: %v", err)
		return
	}
	defer client.CertificatesV1beta1().CertificateSigningRequests().Delete(csr2.Name, nil)

	// testuserClient is a clientset that impersonates 'testUsername', used for
	// testing RBAC rules
	testuserClient := clientset.NewForConfigOrDie(&restclient.Config{
		Host:          s.URL,
		ContentConfig: restclient.ContentConfig{GroupVersion: &schema.GroupVersion{Group: "", Version: "v1"}},
		Impersonate: restclient.ImpersonationConfig{
			UserName: testUsername,
		},
	})

	approvedCSR1 := csr1.DeepCopy()
	approvedCSR1.Status.Conditions = append(approvedCSR1.Status.Conditions, certv1beta1.CertificateSigningRequestCondition{
		Type:    certv1beta1.CertificateApproved,
		Reason:  "AutoApproved",
		Message: "Approved during integration test",
	})
	if _, err := testuserClient.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(approvedCSR1); err != nil {
		t.Errorf("expected to be able to approve CSR1, but request failed: %v", err)
	}

	approvedCSR2 := csr1.DeepCopy()
	approvedCSR2.Status.Conditions = append(approvedCSR2.Status.Conditions, certv1beta1.CertificateSigningRequestCondition{
		Type:    certv1beta1.CertificateApproved,
		Reason:  "AutoApproved",
		Message: "Approved during integration test",
	})
	if _, err := testuserClient.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(approvedCSR2); err == nil {
		t.Errorf("expected approving CSR2 to fail, but it succeeded")
	}
}

func grantUserPermissionToSignFor(client clientset.Interface, username, signerName string) (cleanUp func(), err error) {
	resourceName := "signername-" + username
	cr := buildClusterRoleForSigner(resourceName, signerName)
	crb := buildClusterRoleBindingForUser(resourceName, username, cr.Name)
	if _, err := client.RbacV1().ClusterRoles().Create(cr); err != nil {
		return nil, err
	}
	if _, err := client.RbacV1().ClusterRoleBindings().Create(crb); err != nil {
		return nil, err
	}
	return func() {
		client.RbacV1().ClusterRoles().Delete(cr.Name, nil)
		client.RbacV1().ClusterRoleBindings().Delete(crb.Name, nil)
	}, nil
}

func buildClusterRoleForSigner(name, signerName string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			// must have permission to 'approve' the 'certificatesigners' named
			// 'signerName' to approve CSRs with the given signerName.
			{
				Verbs:         []string{"approve"},
				APIGroups:     []string{certv1beta1.SchemeGroupVersion.Group},
				Resources:     []string{"certificatesigners"},
				ResourceNames: []string{signerName},
			},
			// must have permission to UPDATE the certificatesigningrequests/approval
			// to be able to approve any CSRs at all
			{
				Verbs:     []string{"update"},
				APIGroups: []string{certv1beta1.SchemeGroupVersion.Group},
				Resources: []string{"certificatesigningrequests/approval"},
			},
		},
	}
}

func buildClusterRoleBindingForUser(name, username, clusterRoleName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.UserKind,
				Name: username,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
	}
}

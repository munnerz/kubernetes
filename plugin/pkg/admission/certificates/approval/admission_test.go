package approval

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

type fakeAuthorizer struct{
	t *testing.T
	verb string
	allowedName string
	decision authorizer.Decision
}

func (f fakeAuthorizer) Authorize(ctx context.Context, a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	if a.GetVerb() != f.verb {
		return authorizer.DecisionDeny, fmt.Sprintf("unrecognised verb '%s'", a.GetVerb()), nil
	}
	if a.GetName() != f.allowedName {
		return authorizer.DecisionDeny, fmt.Sprintf("unrecognised name '%s'", a.GetName()), nil
	}
	return f.decision, "", nil
}

func TestIsAuthorizedForPolicy(t *testing.T) {
	tests := map[string]struct{
		signerName string
		allowedName string
		allowed bool
	}{
		"should allow request if user is authorized for specific signerName": {
			signerName: "abc.com/xyz",
			allowedName: "abc.com/xyz",
			allowed: true,
		},
		"should allow request if user is authorized with wildcard": {
			signerName: "abc.com/xyz",
			allowedName: "abc.com/*",
			allowed: true,
		},
		"should deny request if user does not have permission for this signerName": {
			signerName: "abc.com/xyz",
			allowedName: "notabc.com/xyz",
			allowed: false,
		},
	}

	for n, test := range tests {
		t.Run(n, func(t *testing.T) {
			authz := fakeAuthorizer{
				t:           t,
				verb:        "approve",
				allowedName: test.allowedName,
				decision:    authorizer.DecisionAllow,
			}

			allowed := isAuthorizedForPolicy(context.Background(), &user.DefaultInfo{}, "testName", authz)

		})
	}
}

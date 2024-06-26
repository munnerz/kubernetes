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

package filters

import (
	"errors"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	genericrequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/scopes"
	"k8s.io/klog/v2"
	"net/http"
)

const (
	// The 'scope' query parameter in the request URL has an invalid value
	invalidScopeInURL = "invalid scope specified in the request URL"
)

// WithScope extracts the 'scope' query parameter into a context value, if it is specified.
// It avoids resolving the scope to a set of namespaces to allow for authorization on scope names
// to be performed prior to resolution to sets of namespaces.
func WithScope(handler http.Handler, negotiatedSerializer runtime.NegotiatedSerializer) http.Handler {
	return withScope(handler, negotiatedSerializer)
}

// WithScopeResolver resolves the Scope in the request context to a set of namespaces and an identifier for the mapping.
// It then overwrites the Scope value with the new resolved form.
func WithScopeResolver(handler http.Handler, resolver scopes.ScopeResolver) http.Handler {
	return withScopeResolver(handler, resolver)
}

func withScope(handler http.Handler, negotiatedSerializer runtime.NegotiatedSerializer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		info, ok := genericrequest.RequestInfoFrom(ctx)
		if !ok {
			responsewriters.InternalError(w, req, errors.New("no RequestInfo found in the context"))
			return
		}

		var err error
		scope := &scopes.Scope{}
		// extract the user specified scope values
		scope.Name, scope.Value, ok, err = parseScope(req)
		if err != nil {
			gv := schema.GroupVersion{Group: info.APIGroup, Version: info.APIVersion}
			responsewriters.ErrorNegotiated(apierrors.NewBadRequest(err.Error()), negotiatedSerializer, gv, w, req)
			klog.Errorf("Error - %s: %#v", err.Error(), req.RequestURI)
			return
		}
		// fallthrough if no scope parameter specified
		if !ok {
			handler.ServeHTTP(w, req)
			return
		}

		// add the Scope to the context and pass along to the next handler
		req = req.WithContext(scopes.WithScope(ctx, scope))
		handler.ServeHTTP(w, req)
	})
}

func withScopeResolver(handler http.Handler, resolver scopes.ScopeResolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		scope, ok := scopes.ScopeFrom(ctx)
		// fallthrough if no scope parameter specified
		if !ok {
			handler.ServeHTTP(w, req)
			return
		}

		// resolve the scope query parameter to an actual Scope (including currently mapped namespaces)
		scope, err := resolver.Resolve(scope.Name, scope.Value)
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("resolving scope to namespace set: %w", err))
			klog.Errorf("Error - %s: %#v", err.Error(), req.RequestURI)
			return
		}

		// add the Scope to the context and pass along to the next handler
		req = req.WithContext(scopes.WithScope(ctx, scope))
		handler.ServeHTTP(w, req)
	})
}

// parseScope parses the given HTTP request URL and extracts the scope query parameter
// value if specified by the user.
// If a scope is not specified the function returns false
// If the value specified is malformed then the function returns false and err is set
func parseScope(req *http.Request) (string, string, bool, error) {
	value := req.URL.Query().Get("scope")
	if value == "" {
		return "", "", false, nil
	}
	// todo: validate the value is a valid ScopeDefinition name (dns label?)
	// todo: support arbitrary scopeNames?
	scopeName := "internal.apiserver.k8s.io"
	return scopeName, value, true, nil
}

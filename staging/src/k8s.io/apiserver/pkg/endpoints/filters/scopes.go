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
	"fmt"
	"net/http"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/scope"
	"k8s.io/klog/v2"
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
func WithScopeResolver(handler http.Handler, resolver scope.Resolver) http.Handler {
	return withScopeResolver(handler, resolver)
}

func withScope(handler http.Handler, negotiatedSerializer runtime.NegotiatedSerializer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		// extract the user specified scope values
		name, value, urlPath, ok, err := parseScope(req)
		if err != nil {
			responsewriters.ErrorNegotiated(apierrors.NewBadRequest(err.Error()), negotiatedSerializer, meta.SchemeGroupVersion, w, req)
			klog.Errorf("Error - %s: %#v", err.Error(), req.RequestURI)
			return
		}
		// fallthrough if no scope parameter specified
		if !ok {
			handler.ServeHTTP(w, req)
			return
		}
		req.URL.Path = urlPath
		// add the Scope to the context and pass along to the next handler
		req = req.WithContext(scope.WithValue(ctx, scope.NewValue(name, value)))
		handler.ServeHTTP(w, req)
	})
}

func withScopeResolver(handler http.Handler, resolver scope.Resolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		value, ok := scope.ValueFrom(ctx)
		// fallthrough if no scope parameter specified
		if !ok {
			handler.ServeHTTP(w, req)
			return
		}

		// resolve the scope query parameter to an actual Scope (including currently mapped namespaces)
		s, err := resolver.Resolve(ctx, value.Name(), value.Value())
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("resolving scope to namespace set: %w", err))
			klog.Errorf("Error - %s: %#v", err.Error(), req.RequestURI)
			return
		}

		// add the Scope to the context and pass along to the next handler
		req = req.WithContext(scope.WithScope(ctx, s))
		handler.ServeHTTP(w, req)
	})
}

// parseScope parses the given HTTP request URL and extracts the scope query parameter
// value if specified by the user.
// If this request is not for the /scope/ path it returns false.
// Returns a new value for the request path.
// If the value specified is malformed then the function returns false and err is set
func parseScope(req *http.Request) (string, string, string, bool, error) {
	// todo: make this more robust
	if !strings.HasPrefix(req.URL.Path, "/scopes/") {
		return "", "", req.URL.Path, false, nil
	}
	urlPath := strings.SplitN(req.URL.Path, "/", 5)
	if len(urlPath) != 5 {
		return "", "", "", true, fmt.Errorf("invalid scoped request path: %v", req.URL.Path)
	}
	scopeName, scopeValue := urlPath[2], urlPath[3]
	if scopeName == "" {
		return "", "", "", true, fmt.Errorf("no scope name specified")
	}
	if scopeValue == "" {
		return "", "", "", true, fmt.Errorf("no scope value specified")
	}
	// todo: validate the value is a valid ScopeDefinition name (dns label?)
	return scopeName, scopeValue, "/" + urlPath[4], true, nil
}

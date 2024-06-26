package handlers

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/scopes"
	"k8s.io/klog/v2"
)

// TransformScopeSelectors converts field selectors for fields starting 'scopes.k8s.io/' into a labelSelector
// with an IN operator across the list of namespaces within the scope.
// It returns the transformed field and label selector, as well as the minimum permitted resourceVersion
// for requests using this transformation (if one has been applied).
// todo: once Create requests have LabelSelectors, we should verify that the namespace is still a part of the scope the object is being written for if we want to ensure no writes to objects outside of scopes can happen through scoped endpoints.
func TransformScopeSelectors(ctx context.Context, mgr scopes.Manager, fieldSelector fields.Selector, labelSelector labels.Selector) (fields.Selector, labels.Selector, uint64, error) {
	// Fast path if no FieldSelector is specified.
	if fieldSelector == nil {
		return fieldSelector, labelSelector, 0, nil
	}
	// Extract any scopes.k8s.io/* field selectors from the request first.
	var err error
	var scopeName, scopeValue string
	if fieldSelector, err = fieldSelector.Transform(func(field, value string) (newField, newValue string, err error) {
		prefix, name, found := strings.Cut(field, "/")
		if !found || prefix != "scopes.k8s.io" {
			// only include keys that don't start with 'scopes.k8s.io/'
			return field, value, nil
		}
		if scopeName != "" || scopeValue != "" {
			return "", "", errors.NewBadRequest("cannot specify more than one scopes.k8s.io label selector")
		}
		scopeValue = value
		scopeName = name
		// remove this field selector from the request as it will be replaced with a label selector
		return "", "", nil
	}); err != nil {
		return nil, nil, 0, err
	}
	// If no scopes.k8s.io/ prefixed field selector is present, continue like normal
	if scopeName == "" {
		return fieldSelector, labelSelector, 0, nil
	}
	// Lookup current set of Namespaces with this scope label set on them with a consistent read.
	namespaces, minimumRV, err := mgr.GetNamespaces(ctx, scopeName, scopeValue)
	if err != nil {
		// todo: consider the error type here, do we want to return ResourceVersionTooOld?
		return nil, nil, 0, errors.NewInternalError(err)
	}
	// Overwrite the LabelSelector with the scope->namespace set transformation applied
	if labelSelector == nil {
		labelSelector = labels.NewSelector()
	}
	req, err := labels.NewRequirement("metadata.namespace", selection.In, namespaces)
	if err != nil {
		return nil, nil, 0, errors.NewInternalError(err)
	}
	labelSelector = labelSelector.Add(*req)
	return fieldSelector, labelSelector, minimumRV, nil
}

// TransformScopedResourceVersion transforms a ResourceVersion embedded in an object received from a client during a
// write operation into a 'standard' non-generational ResourceVersion.
func TransformScopedResourceVersion(objectMeta metav1.Object) error {
	// If the first digit of the resourceVersion string in the object body is zero, this request is from
	// a client that is 'scoped'. We must use this trick to determine if this is a scoped request as
	// label selectors are not permitted in create requests at this time.
	rv := objectMeta.GetResourceVersion()
	if len(rv) == 0 || rv[0] != '0' {
		return nil
	}
	_, rv, found := strings.Cut(rv, "9")
	if !found {
		klog.Warningf("Found object that is expected to be a scoped request that does not contain the generation marker: %q", rv)
		return nil
	}
	objectMeta.SetResourceVersion(rv)
	return nil
}

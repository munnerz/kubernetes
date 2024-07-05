package handlers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/scope"
)

var (
	namespaceGroupResource = corev1.Resource("namespaces")
)

func scopingLabelSelector(s scope.Scope, resource schema.GroupResource, selector labels.Selector) (labels.Selector, error) {
	if selector == nil {
		selector = labels.NewSelector()
	}
	key := "kubernetes.io/metadata.namespace"
	if resource == namespaceGroupResource {
		key = "kubernetes.io/metadata.name"
	}
	req, err := labels.NewRequirement(key, selection.In, s.Namespaces())
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	selector = selector.Add(*req)
	return selector, nil
}

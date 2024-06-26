package handlers

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/scopes"
)

func scopingLabelSelector(s *scopes.Scope, selector labels.Selector) (labels.Selector, error) {
	if selector == nil {
		selector = labels.NewSelector()
	}
	req, err := labels.NewRequirement("kubernetes.io/metadata.namespace", selection.In, s.Namespaces)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	selector = selector.Add(*req)
	return selector, nil
}

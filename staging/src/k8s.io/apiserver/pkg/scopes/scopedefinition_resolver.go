package scopes

import (
	"k8s.io/client-go/informers"
	scopeslisters "k8s.io/client-go/listers/scopes/v1alpha1"
)

// NewScopeDefinitionResolver creates a new ScopeResolver that resolves scopes to a list of
// namespaces by querying a lister for the currently configured mapping.
func NewScopeDefinitionResolver(factory informers.SharedInformerFactory) ScopeResolver {
	return &scopeDefinitionResolver{
		lister: factory.Scopes().V1alpha1().ScopeDefinitions().Lister(),
	}
}

// scopeDefinitionResolver resolves scopes by querying a lister for
type scopeDefinitionResolver struct {
	lister scopeslisters.ScopeDefinitionLister
}

func (s scopeDefinitionResolver) Resolve(name, value string) (*Scope, error) {
	defName := name + ":" + value
	def, err := s.lister.Get(defName)
	if err != nil {
		return nil, err
	}
	return &Scope{
		Name:       name,
		Value:      value,
		Namespaces: def.Spec.Namespaces,
		Identifier: def.ResourceVersion,
	}, nil
}

var _ ScopeResolver = &scopeDefinitionResolver{}

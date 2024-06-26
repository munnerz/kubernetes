package scopes

// Scope is an internal representation of a scope->namespaces mapping.
// It carries an identifier for the mapping, which can be used to verify whether a mapping
// is still valid when handling requests.
type Scope struct {
	// Name and value of the scope used to resolve these namespaces
	Name, Value string

	// Namespaces included within the scope.
	Namespaces []string

	// Identifier used to indicate the version of this mapping for the scope.
	Identifier string
}

// ScopeResolver can resolve a scope name and value pair into a list of namespaces.
// It returns an epoch identifier which should be treated opaquely to
type ScopeResolver interface {
	Resolve(name, value string) (*Scope, error)
}

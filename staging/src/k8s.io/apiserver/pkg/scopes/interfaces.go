package scopes

// ScopeResolver can resolve a scope name and value pair into a list of namespaces.
// It returns an epoch identifier which should be treated opaquely to
type ScopeResolver interface {
	// Resolve converts a (name, value) pair for a scope into a Scope object using
	// an implementation specific storage.
	Resolve(name, value string) (*Scope, error)
}

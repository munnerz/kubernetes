package scope

import "context"

// Resolver can resolve a scope name and value pair into a list of namespaces.
// It returns an epoch identifier which should be treated opaquely to
type Resolver interface {
	// Resolve converts a (name, value) pair for a scope into a Scope object using
	// an implementation specific storage.
	Resolve(ctx context.Context, name, value string) (Scope, error)
}

// ScopeValue is a pointer to a named mapping from (name, value)->Scope.
type ScopeValue interface {
	Name() string
	Value() string
}

type Scope interface {
	// ScopeValue contains the (Name, Value) tuple identifying the scope.
	ScopeValue

	// Identifier is used to identify this specific iteration of the scope.
	// It acts as a 'generation' of the resolved set of namespaces, which may change when
	// the scope mapping is updated.
	Identifier() string

	// Namespaces returns the list of namespaces included in this scope.
	Namespaces() []string

	// Expired returns a channel that is used to determine whether this Scope mapping is still valid.
	Expired() <-chan struct{}
	// Err details why the scope has expired.
	Err() error
}

package scopes

import "k8s.io/apimachinery/pkg/runtime/schema"

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

	// all set by the resolver when the Scope is constructed
	minimumResourceVersion func(*Scope, schema.GroupResource) (uint64, error)
	expired                bool
	expiredCh              chan struct{}
}

func (s *Scope) DefinitionName() string {
	return s.Name + ":" + s.Value
}

// ExpiredChan can be read from to determine if this scope is still valid.
// The channel will be closed once this Scope can no longer be considered valid.
func (s *Scope) ExpiredChan() <-chan struct{} {
	return s.expiredCh
}

// MinimumResourceVersion returns the minimum supported starting resource version for this scope.
// This is used to identify watches that **began** prior to all apiservers acknowledging a mapping
// configuration so they can be forced to re-list to ensure consistency.
func (s *Scope) MinimumResourceVersion(resource schema.GroupResource) (uint64, error) {
	return s.minimumResourceVersion(s, resource)
}

// expire is not safe to be called from multiple goroutines
func (s *Scope) expire() {
	if s.expired {
		return
	}
	s.expired = true
	close(s.expiredCh)
}

// Expired returns true if the Scope can no longer be considered valid and any users of it must reset their state.
// This typically happens when a new scope generation has replaced an old one.
// If the ExpiredCh is not set, this function returns true (i.e. 'expired').
func Expired(s *Scope) bool {
	select {
	case <-s.ExpiredChan():
		return true
	default:
		return false
	}
}

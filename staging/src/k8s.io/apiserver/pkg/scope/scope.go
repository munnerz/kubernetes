package scope

// value implements Values
type value struct {
	name, value string
}

// NewValue returns a Scope holding only a (name, value) tuple referring to a particular scope.
func NewValue(name, val string) ScopeValue {
	return &value{
		name:  name,
		value: val,
	}
}

// Name returns the name portion of the scope, e.g. 'workspace'.
func (s *value) Name() string {
	return s.name
}

// Value returns the value portion of the scope, e.g. 'my-workspace'.
func (s *value) Value() string {
	return s.value
}

// ObjectName returns the constructed Scope name for a given Value.
func ObjectName(s ScopeValue) string {
	return s.Name() + ":" + s.Value()
}

// NewScope returns a new Scope that is always considered expired.
// It's Expired method will return a nil channel so any callers that attempt to rely
// on the freshness of the mapping will assume the channel is already closed.
func newScope(val ScopeValue, identifier string, namespaces []string) *scope {
	return &scope{
		ScopeValue: val,
		namespaces: namespaces,
		identifier: identifier,
	}
}

// Scope is an internal representation of a scope->namespaces mapping.
// It carries an identifier for the mapping, which can be used to verify whether a mapping
// is still valid when handling requests.
type scope struct {
	ScopeValue

	// namespaces included within the scope.
	namespaces []string

	// Identifier used to indicate the version of this mapping for the scope.
	identifier string

	// current is true if this scope is not considered expired.
	// we don't use 'expired' here so that the default value (false) is consistent
	// with the behaviour of the default value of expiredCh (nil).
	current   bool
	err       error
	expiredCh *chan struct{}
}

func (s *scope) Identifier() string {
	return s.identifier
}

func (s *scope) Namespaces() []string {
	return s.namespaces
}

func (s *scope) Expired() <-chan struct{} {
	if s.expiredCh == nil {
		return nil
	}
	return *s.expiredCh
}

func (s *scope) Err() error {
	return s.err
}

func (s *scope) expire(err error) {
	if !s.current {
		return
	}
	s.current = false
	s.err = err
	close(*s.expiredCh)
}

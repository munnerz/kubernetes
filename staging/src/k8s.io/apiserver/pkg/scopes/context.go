package scopes

import "context"

type scopeKeyType int

// scopeKey is the Scope key for the context. It's of private type here. Because
// keys are interfaces and interfaces are equal when the type and the value is equal, this
// does not conflict with other keys like it.
const scopeKey scopeKeyType = iota

// WithScope returns a copy of parent in which the scope value is set

func WithScope(ctx context.Context, s *Scope) context.Context {
	return context.WithValue(ctx, scopeKey, s)
}

// ScopeFrom returns the value of the Scope key on the ctx
func ScopeFrom(ctx context.Context) (*Scope, bool) {
	info, ok := ctx.Value(scopeKey).(*Scope)
	return info, ok
}

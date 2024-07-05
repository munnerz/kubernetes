package scope

import "context"

type scopeKeyType int

// scopeKey is the Scope key for the context. It's of private type here. Because
// keys are interfaces and interfaces are equal when the type and the value is equal, this
// does not conflict with other keys like it.
const scopeKey scopeKeyType = iota

// WithScope returns a copy of parent in which the Scope is set.
// It shares the same context key as WithValue for convenient access.
func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeKey, s)
}

// WithValue returns a copy of parent in which the Value is set.
// It shares the same context key as WithScope for convenient access.
func WithValue(ctx context.Context, s ScopeValue) context.Context {
	return context.WithValue(ctx, scopeKey, s)
}

// ScopeFrom returns the value of the Scope key on the ctx
func ScopeFrom(ctx context.Context) (Scope, bool) {
	info, ok := ctx.Value(scopeKey).(Scope)
	return info, ok
}

func ValueFrom(ctx context.Context) (ScopeValue, bool) {
	info, ok := ctx.Value(scopeKey).(ScopeValue)
	return info, ok
}

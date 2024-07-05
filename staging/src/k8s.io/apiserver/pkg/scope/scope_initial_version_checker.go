package scope

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"time"

	"k8s.io/apimachinery/pkg/watch"
)

type MinimumScopeVersionChecker interface {
	// Cancel will stop the version checker running and cause the go routine to exit.
	// The checker cannot be re-used after Cancel is called.
	Cancel()
}

func NewMinimumScopeVersionChecker(ctx context.Context, interval time.Duration, scope *Scope, w watch.Interface, initialRV uint64) MinimumScopeVersionChecker {
	ctx, cancel := context.WithCancelCause(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:

			case <-ctx.Done():
				cancel(ctx.Err())
				return
			}
		}
	}()
	return nil
}

type msvChecker struct {
	cancel func()

	///

	// all set by the resolver when the Scope is constructed

	// MinimumResourceVersion returns the minimum supported starting resource version for this scope.
	// This is used to identify watches that **began** prior to all apiservers acknowledging a mapping
	// configuration so they can be forced to re-list to ensure consistency.
	minimumResourceVersion func(Scope, schema.GroupResource) (uint64, error)
}

func (c *msvChecker) Cancel() {
	c.cancel()
}

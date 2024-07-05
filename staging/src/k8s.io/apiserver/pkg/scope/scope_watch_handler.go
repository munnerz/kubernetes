package scope

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type WatchHandler interface {
	Scope

	SetInitialResourceVersion(uint64) error
	SetMinimumResourceVersion(uint64) error
}

const (
	minimumResourceVersionUpdateChannelSize = 10
	initialResourceVersionUpdateGracePeriod = time.Millisecond * 100
)

var minimumResourceVersionErr = errors.New("starting resource version is no longer valid")

type scopeWatchHandler struct {
	parent Scope
	*scope

	minimumResourceVersion     chan uint64
	initialResourceVersionChan chan uint64
}

func NewScopeWatchHandler(ctx context.Context, parent Scope) WatchHandler {
	// create a new watch handler
	wh := &scopeWatchHandler{
		parent: parent,
		// create a new scope so we can re-use the expiration handling
		scope: &scope{
			ScopeValue: NewValue(parent.Name(), parent.Value()),
			identifier: parent.Identifier(),
			namespaces: parent.Namespaces(),
			current:    true,
			expiredCh:  new(chan struct{}),
		},
		minimumResourceVersion:     make(chan uint64, minimumResourceVersionUpdateChannelSize),
		initialResourceVersionChan: make(chan uint64),
	}
	go wh.waitForMinimumResourceVersion(ctx.Done())
	return wh
}

// SetMinimumResourceVersion updates the minimum allowed initial starting resource version.
// This may cause the Scope to be expired if the initial resource version is less than the minimum.
func (wh *scopeWatchHandler) SetMinimumResourceVersion(rv uint64) error {
	select {
	case wh.minimumResourceVersion <- rv:
		return nil
	default:
		return fmt.Errorf("scope watch handler processing blocked, minimum resource version not acknowledged")
	}
}

// SetInitialResourceVersion may only be called once.
// After the initial call, it will always return the same value.
func (wh *scopeWatchHandler) SetInitialResourceVersion(rv uint64) error {
	return sync.OnceValue(func() error {
		select {
		case wh.initialResourceVersionChan <- rv:
			return nil
		default:
			// just in case the goroutine for this handler has not started yet when this is called,
			// allow for 100ms here just in case things are running slow.. this would be
			// considered a weird edge-case, but it avoids us blocking forever whilst being permissive
			// when things are slow.
		}
		t := time.NewTicker(initialResourceVersionUpdateGracePeriod)
		defer t.Stop()
		select {
		case wh.initialResourceVersionChan <- rv:
			return nil
		case <-t.C:
			return errors.New("initial resource version already set in scoped watch handler")
		}
	})()
}

func (wh *scopeWatchHandler) waitForMinimumResourceVersion(done <-chan struct{}) (err error) {
	// ensure we always eventually expire so we don't leak the goroutine
	defer wh.expire(err)
	defer close(wh.minimumResourceVersion)

	initialRV, err := wh.readInitialResourceVersion(done)
	if err != nil {
		return err
	}

	for {
		select {
		case <-done:
			return nil
		case <-wh.parent.Expired():
			return wh.parent.Err()
		case minimumRV := <-wh.minimumResourceVersion:
			if initialRV < minimumRV {
				return apierrors.NewResourceExpired(fmt.Sprintf("initial resourceVersion %q is older than minimum resourceVersion %q", initialRV, minimumRV))
			}
			// nothing to do if the initialRV is less than the minimumRV so we just go to the next loop
		}
	}
}

func (wh *scopeWatchHandler) readInitialResourceVersion(done <-chan struct{}) (uint64, error) {
	// we only allow one initialResourceVersion to be set
	defer close(wh.initialResourceVersionChan)
	select {
	case <-done:
		return 0, errors.New("done channel closed")
	case <-wh.parent.Expired():
		return 0, wh.parent.Err()
	case rv, ok := <-wh.initialResourceVersionChan:
		if !ok {
			return 0, errors.New("no initial resource version found")
		}
		return rv, nil
	}
}

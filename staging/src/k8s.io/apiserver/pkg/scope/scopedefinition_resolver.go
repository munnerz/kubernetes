package scope

import (
	"context"
	"errors"
	"fmt"
	scopesv1alpha1 "k8s.io/api/scopes/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/storage"
	scopesinformers "k8s.io/client-go/informers/scopes/v1alpha1"
	"k8s.io/client-go/kubernetes"
	scopeslisters "k8s.io/client-go/listers/scopes/v1alpha1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	controllerName = "request_scope_resolver"
)

// NewScopeDefinitionResolver creates a new Resolver that resolves scopes to a list of
// namespaces by querying a lister for the currently configured mapping.
func NewScopeDefinitionResolver(apiServerID string, storeMapper ResourceStoreMapper, clientset kubernetes.Interface, scopeDefinitionInformer scopesinformers.ScopeDefinitionInformer) (*DefaultScopeResolver, error) {
	r := &DefaultScopeResolver{
		apiServerID:           apiServerID,
		scopes:                make(map[string]map[string]*scope),
		clientset:             clientset,
		lister:                scopeDefinitionInformer.Lister(),
		storeMapper:           storeMapper,
		scopeDefinitionSynced: scopeDefinitionInformer.Informer().HasSynced,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{
				Name: controllerName,
			},
		),
	}
	scopeDefinitionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: r.enqueue,
		UpdateFunc: func(_, newObj interface{}) {
			r.enqueue(newObj)
		},
		DeleteFunc: nil, // todo - cancel all existing contexts and cleanup
	})

	return r, nil
}

// ResourceStoreMapper knows how to map group/resource pairs to a particular storage backend (i.e. etcd).
// It returns an opaque string identifier (the 'storeID') for the storage backend that the resource is
// known to be stored within.
type ResourceStoreMapper interface {
	// StoreForResource returns an opaque string identifier for the storage cluster (aka etcd cluster)
	// that this resource is stored in.
	StoreForResource(schema.GroupResource) (string, error)

	// Stores returns the complete list of stores that this apiserver may use to store data.
	Stores() []string

	// CurrentResourceVersion returns the current resource version of the store with the given identifier.
	CurrentResourceVersion(ctx context.Context, storeID string) (uint64, error)
}

// DefaultScopeResolver resolves scopes by querying a lister for
type DefaultScopeResolver struct {
	// apiServerID is the identifier for this apiserver
	apiServerID string

	// map of scopeName->scopeValue->*scope
	scopesLock sync.RWMutex
	scopes     map[string]map[string]*scope

	clientset   kubernetes.Interface
	storeMapper ResourceStoreMapper
	versioner   storage.Versioner

	// controller related fields
	lister                scopeslisters.ScopeDefinitionLister
	scopeDefinitionSynced cache.InformerSynced
	queue                 workqueue.TypedRateLimitingInterface[string]
}

// Resolve converts a (name, value) pair for a scope into a Scope object by reading the scope
// from the internally maintained mapping, mirrored from ScopeDefinition objects.
func (r *DefaultScopeResolver) Resolve(ctx context.Context, name, value string) (Scope, error) {

	return r.readScope(name, value)
}

// enqueue expects to be called with ScopeDefinition objects
func (r *DefaultScopeResolver) enqueue(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	r.queue.Add(key)
}

// Run starts one worker.
func (r *DefaultScopeResolver) Run(ctx context.Context) {
	logger := klog.FromContext(ctx)
	defer utilruntime.HandleCrash()
	defer r.queue.ShutDown()
	defer logger.Info("Shutting down scope resolver controller")

	logger.Info("Starting scope resolver controller")

	if !cache.WaitForCacheSync(ctx.Done(), r.scopeDefinitionSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	// only run one worker
	go wait.UntilWithContext(ctx, r.runWorker, time.Second)

	<-ctx.Done()
}

func (r *DefaultScopeResolver) runWorker(ctx context.Context) {
	for r.processNextWorkItem(ctx) {
	}
}

func (r *DefaultScopeResolver) processNextWorkItem(ctx context.Context) bool {
	key, quit := r.queue.Get()
	if quit {
		return false
	}
	// Make sure we acknowledge this queue item as Done at the very least.
	defer r.queue.Done(key)
	if err := r.process(ctx, key); err != nil {
		utilruntime.HandleError(err)
		r.queue.AddRateLimited(key)
		return true
	}
	r.queue.Forget(key)
	return true
}

// process is NOT safe for concurrent execution
func (r *DefaultScopeResolver) process(ctx context.Context, key string) error {
	def, err := r.lister.Get(key)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	logger := klog.FromContext(ctx).WithValues("scopedefinition", def, "scope_id", def.Status.ScopeID)
	parts := strings.SplitN(def.Name, ":", 2)
	scopeName, scopeValue := parts[0], parts[1]
	// check if we already have an entry for this scope name
	// don't need to acquire an RLock as we are the only writer
	if _, ok := r.scopes[scopeName]; !ok {
		// no race as handle() is always single-threaded
		r.writeScope(scopeName, scopeValue, def)
		logger.V(4).Info("New Scope configuration installed")
		return nil
	}
	// check if we already have an entry for this scope value
	// don't need to acquire an RLock as we are the only writer
	existingScope, ok := r.scopes[scopeName][scopeValue]
	if !ok {
		r.writeScope(scopeName, scopeValue, def)
		logger.V(4).Info("New Scope configuration installed")
		return nil
	}
	// check if we need to update the scope (and expire the old one)
	// don't need to acquire an RLock as we are the only writer
	if existingScope.Identifier() == def.Status.ScopeID {
		// the identifier has not changed, so we have nothing more to do
		return nil
	}
	// set a new scope in its place so new requests use the new mapping
	newScope := r.writeScope(scopeName, scopeValue, def)
	// expire the old Scope and set the new one
	existingScope.expire(errors.New("scope configuration changed"))
	// todo: wait for all watchers to stop sending events?
	// deep copy as we are about to modify the status.serverScopeVersions field
	def = def.DeepCopy()
	// fetch the current resource version for all stores
	for _, store := range r.storeMapper.Stores() {
		// todo: run in parallel
		rv, err := r.storeMapper.CurrentResourceVersion(ctx, store)
		if err != nil {
			// if an error occurs, clear the existing stored scope, expire the new one and retry
			r.writeScope(scopeName, scopeValue, nil)
			// this message is tailored to existing 'watch' users so contains less detail
			newScope.expire(fmt.Errorf("internal error"))
			return fmt.Errorf("failed to fetch resourceVersion for store %q: %w", store, err)
		}
		SetServerScopeVersion(&def.Status, scopesv1alpha1.ServerScopeVersion{
			APIServerID:     r.apiServerID,
			StoreID:         store,
			ScopeID:         newScope.Identifier(),
			ResourceVersion: strconv.FormatUint(rv, 10),
		})
	}
	if _, err := r.clientset.ScopesV1alpha1().ScopeDefinitions().UpdateStatus(ctx, def, metav1.UpdateOptions{}); err != nil {
		// if an error occurs, clear the existing stored scope, expire the new one and retry
		r.writeScope(scopeName, scopeValue, nil)
		// this message is tailored to existing 'watch' users so contains less detail
		newScope.expire(fmt.Errorf("internal error"))
		return fmt.Errorf("failed to update status for scopedefinition %q: %w", def.Name, err)
	}
	logger.V(4).Info("Updated Scope configuration", "old_scope_id", existingScope.Identifier())
	return nil
}

func (r *DefaultScopeResolver) writeScope(name, value string, def *scopesv1alpha1.ScopeDefinition) *scope {
	r.scopesLock.Lock()
	defer r.scopesLock.Unlock()
	clear := def == nil
	// the new scope to be set in the store
	expiringScope := r.buildExpiringScope(def)
	if _, ok := r.scopes[name]; !ok {
		if clear {
			return nil
		}
		r.scopes[name] = make(map[string]*scope)
	}
	if clear {
		delete(r.scopes[name], value)
		return nil
	}
	r.scopes[name][value] = expiringScope
	return expiringScope
}

func (r *DefaultScopeResolver) readScope(name, value string) (*scope, error) {
	r.scopesLock.RLock()
	defer r.scopesLock.RUnlock()
	values, ok := r.scopes[name]
	if !ok {
		return nil, fmt.Errorf("unknown scope %q", name)
	}
	scope, ok := values[value]
	if !ok {
		return nil, fmt.Errorf("unknown scope value '%s=%s'", name, value)
	}
	return scope, nil
}

func (r *DefaultScopeResolver) buildExpiringScope(def *scopesv1alpha1.ScopeDefinition) *scope {
	if def == nil {
		return nil
	}
	parts := strings.SplitN(def.Name, ":", 2)
	scopeName, scopeValue := parts[0], parts[1]
	return &scope{
		ScopeValue: NewValue(scopeName, scopeValue),
		namespaces: def.Status.Namespaces,
		identifier: def.Status.ScopeID,
		current:    true,
		expiredCh:  new(chan struct{}),
	}
}

func (r *DefaultScopeResolver) minimumResourceVersion(scope Scope, resource schema.GroupResource) (uint64, error) {
	// todo: optimise this function to return as quick as possible as it will be called very frequently
	storeID, err := r.storeMapper.StoreForResource(resource)
	if err != nil {
		return 0, err
	}
	def, err := r.lister.Get(DefinitionName(scope))
	if err != nil {
		return 0, err
	}
	for _, mrv := range def.Status.MinimumResourceVersions {
		if mrv.StoreID != storeID {
			continue
		}
		// todo: cache this parsing/entire result
		rv, err := r.versioner.ParseResourceVersion(mrv.ResourceVersion)
		if err != nil {
			return 0, err
		}
		return rv, nil
	}
	// todo: wait some small number of seconds and retry, as this may resolve itself very soon?
	return 0, fmt.Errorf("no minimum resource version entry for store %q found", storeID)
}

var _ Resolver = &DefaultScopeResolver{}

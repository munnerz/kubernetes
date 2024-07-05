package requestscoping

import (
	"context"
	"fmt"
	"sort"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	scopesinformers "k8s.io/client-go/informers/scopes/v1alpha1"
	"k8s.io/client-go/kubernetes"
	scopeslisters "k8s.io/client-go/listers/scopes/v1alpha1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller watches Scope objects and updates their status accordingly.
type Controller struct {
	clientset kubernetes.Interface

	scopeDefinitionLister scopeslisters.ScopeLister
	scopeDefinitionSynced cache.InformerSynced

	queue workqueue.TypedRateLimitingInterface[string]
}

// NewScopeController creates a new Controller that handles computing and updating Scope objects.
func NewScopeController(ctx context.Context, clientset kubernetes.Interface, scopeInformer scopesinformers.ScopeInformer) *Controller {
	c := &Controller{
		clientset:             clientset,
		scopeDefinitionLister: scopeInformer.Lister(),
		scopeDefinitionSynced: scopeInformer.Informer().HasSynced,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{
				Name: "scope_controller",
			},
		),
	}
	scopeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(_, newObj interface{}) {
			c.enqueue(newObj)
		},
		DeleteFunc: nil, // todo
	})
	return c
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

// Run starts one worker.
func (c *Controller) Run(ctx context.Context) {
	logger := klog.FromContext(ctx)
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	defer logger.Info("Shutting down scope definition controller")

	logger.Info("Starting storage version garbage collector")

	if !cache.WaitForCacheSync(ctx.Done(), c.scopeDefinitionSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	go wait.UntilWithContext(ctx, c.runWorker, time.Second)

	<-ctx.Done()
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Make sure we acknowledge this queue item as Done at the very least.
	defer c.queue.Done(key)
	if err := c.process(ctx, key); err != nil {
		utilruntime.HandleError(err)
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

func (c *Controller) process(ctx context.Context, key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// avoid re-queuing bad keys
		utilruntime.HandleError(err)
		return nil
	}
	def, err := c.scopeDefinitionLister.Get(name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	logger := klog.FromContext(ctx).WithValues("scope", klog.KObj(def))
	proposedSet := sets.New(def.Spec.Namespaces...)
	currentSet := sets.New(def.Status.Namespaces...)
	if proposedSet.Equal(currentSet) {
		logger.V(6).Info("No change detected")
		// nothing to do, status already up to date
		return nil
	}
	// sort the list to ensure it is stable when changes are made
	newNamespaces := proposedSet.UnsortedList()
	sort.Strings(newNamespaces)
	// perform a deep-copy as we don't want to mutate cache items
	def = def.DeepCopy()
	def.Status.Namespaces = newNamespaces
	if _, err := c.clientset.ScopesV1alpha1().Scopes().UpdateStatus(ctx, def, metav1.UpdateOptions{}); err != nil {
		return err
	}
	logger.V(4).Info("Scope updated with new status.namespaces configuration")
	return nil
}

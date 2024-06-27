package options

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"golang.org/x/exp/maps"

	scopesv1alpha1 "k8s.io/api/scopes/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	genericfeatures "k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/scopes"
	"k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	cliflag "k8s.io/component-base/cli/flag"
	scopesinternal "k8s.io/kubernetes/pkg/apis/scopes"
)

type RequestScopingOptions struct {
	// RemoteKubeConfigFile is the file to use to connect to a "normal" kube API server which hosts the
	// ScopeDefinition.scopes.k8s.io endpoint for reading scope configurations.
	RemoteKubeConfigFile string

	ResourceStoreID         string
	OverrideResourceStoreID cliflag.ConfigurationMap
}

func NewRequestScopingOptions() *RequestScopingOptions {
	return &RequestScopingOptions{}
}

func (s *RequestScopingOptions) Validate() []error {
	if s == nil {
		return nil
	}

	allErrors := []error{}
	if !utilfeature.DefaultFeatureGate.Enabled(genericfeatures.RequestScoping) {
		if len(s.RemoteKubeConfigFile) > 0 {
			allErrors = append(allErrors, fmt.Errorf("--request-scope-kubeconfig requires the %q feature gate", genericfeatures.RequestScoping))
		}
		if len(s.ResourceStoreID) > 0 {
			allErrors = append(allErrors, fmt.Errorf("--resource-store-id requires the %q feature gate", genericfeatures.RequestScoping))
		}
		if len(s.OverrideResourceStoreID) > 0 {
			allErrors = append(allErrors, fmt.Errorf("--resource-store-id-overrides requires the %q feature gate", genericfeatures.RequestScoping))
		}
		return allErrors
	}

	if s.ResourceStoreID == "" {
		allErrors = append(allErrors, fmt.Errorf("--resource-store-id must be specified"))
	}
	// todo: validate all resources with etcd server overrides have an override set
	return allErrors
}

func (s *RequestScopingOptions) AddFlags(fs *pflag.FlagSet) {
	if s == nil {
		return
	}

	fs.StringVar(&s.RemoteKubeConfigFile, "request-scope-kubeconfig", s.RemoteKubeConfigFile, ""+
		"kubeconfig file pointing at the 'core' kubernetes server with enough rights to read and update "+
		"scopedefinitions.scopes.k8s.io. (requires '"+string(genericfeatures.RequestScoping)+"' feature gate")
	fs.StringVar(&s.ResourceStoreID, "resource-store-id", s.ResourceStoreID, ""+
		"identifier for the storage backend for resources which must be unique per set of aggregated apiservers")
	fs.Var(&s.OverrideResourceStoreID, "override-resource-store-id", ""+
		"A set of key=value pairs that set the store IDs for resources with etcd storage overrides. Use as:\n"+
		"events.k8s.io/events=<unique identifier for events storage>\n")
}

func (s *RequestScopingOptions) ApplyTo(config *server.Config, loopbackConfig *rest.Config, storageFactory serverstorage.StorageFactory) error {
	if !utilfeature.DefaultFeatureGate.Enabled(genericfeatures.RequestScoping) {
		return nil
	}
	if s == nil {
		config.ScopeResolver = nil
		return nil
	}

	// build a mapper that is able to map GroupResource->an identifier for the etcd store
	overrides := make(map[schema.GroupResource]string)
	for groupResource, override := range s.OverrideResourceStoreID {
		resource := schema.ParseGroupResource(groupResource)
		overrides[resource] = override
	}
	storeMapper, err := newSimpleStoreMapper(s.ResourceStoreID, overrides, storageFactory)

	// build a client configure with either the loopback or the explicitly provided request-scope-kubeconfig
	client, err := s.getClient(loopbackConfig)
	if err != nil {
		return fmt.Errorf("failed to get delegated scope configuration kubeconfig: %v", err)
	}
	versionedInformers := informers.NewSharedInformerFactory(client, 0)

	// build the resolver
	scopeResolver, err := scopes.NewScopeDefinitionResolver(config.APIServerID, storeMapper, client, versionedInformers.Scopes().V1alpha1().ScopeDefinitions())
	if err != nil {
		return fmt.Errorf("failed building scope resolver: %w", err)
	}

	// register a PostStart hook to start the informers and resolver
	postStartHook := func(context server.PostStartHookContext) error {
		versionedInformers.Start(context.Context.Done())
		go scopeResolver.Run(context.Context)
		return nil
	}
	config.ScopeResolver = scopeResolver
	err = config.AddPostStartHook("scopes/start-scope-resolver", postStartHook)
	if err != nil {
		return fmt.Errorf("failed to add post start hook for scope resolver: %w", err)
	}

	return nil
}

// getClient returns a Kubernetes clientset.
// If RemoteKubeConfigFile is not set, a client using the loopback config will be used.
func (s *RequestScopingOptions) getClient(clientConfig *rest.Config) (kubernetes.Interface, error) {
	var err error
	if len(s.RemoteKubeConfigFile) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: s.RemoteKubeConfigFile}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		clientConfig, err = loader.ClientConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// perform a copy as we are going to modify it below
		clientConfig = rest.CopyConfig(clientConfig)
	}

	// set high qps/burst limits since this will effectively limit API server responsiveness
	clientConfig.QPS = 200
	clientConfig.Burst = 400
	// do not set a timeout on the http client, instead use context for cancellation
	// if multiple timeouts were set, the request will pick the smaller timeout to be applied, leaving other useless.
	//
	// see https://github.com/golang/go/blob/a937729c2c2f6950a32bc5cd0f5b88700882f078/src/net/http/client.go#L364
	// todo: explore using this here too
	//if s.CustomRoundTripperFn != nil {
	//	clientConfig.Wrap(s.CustomRoundTripperFn)
	//}

	return kubernetes.NewForConfig(clientConfig)
}

const emptyResourcePrefix = "/_empty"

var scopeDefinitionResource = scopesv1alpha1.Resource("scopedefinitions")

func newScopeDefinition() runtime.Object     { return &scopesinternal.ScopeDefinition{} }
func newScopeDefinitionList() runtime.Object { return &scopesinternal.ScopeDefinitionList{} }

func newSimpleStoreMapper(defaultStoreID string, overrides map[schema.GroupResource]string, storageFactory serverstorage.StorageFactory) (scopes.ResourceStoreMapper, error) {
	storeConfigs := map[string]*storagebackend.ConfigForResource{
		defaultStoreID: (storageFactory.Configs()[0]).ForResource(scopeDefinitionResource),
	}
	for resource, storeID := range overrides {
		config, err := storageFactory.NewConfig(resource, nil)
		if err != nil {
			return nil, err
		}
		storeConfigs[storeID] = config.ForResource(scopeDefinitionResource)
	}

	stores := make(map[string]storage.Interface)
	var stops []func()
	for storeID, config := range storeConfigs {
		store, stop, err := factory.Create(*config, newScopeDefinition, newScopeDefinitionList, emptyResourcePrefix)
		if err != nil {
			return nil, err
		}
		stops = append(stops, stop)
		stores[storeID] = store
	}
	return &simpleStoreMapper{
		defaultStoreID: defaultStoreID,
		overrides:      overrides,
		stores:         stores,
	}, nil
}

type simpleStoreMapper struct {
	defaultStoreID string
	overrides      map[schema.GroupResource]string

	stores map[string]storage.Interface
	stop   func()
}

func (r *simpleStoreMapper) Stores() []string {
	return sets.New(r.defaultStoreID).Insert(maps.Values(r.overrides)...).UnsortedList()
}

func (r *simpleStoreMapper) CurrentResourceVersion(ctx context.Context, storeID string) (uint64, error) {
	store, ok := r.stores[storeID]
	if !ok {
		return 0, fmt.Errorf("unrecognised store ID %q", storeID)
	}
	return storage.GetCurrentResourceVersionFromStorage(ctx, store, newScopeDefinitionList, emptyResourcePrefix, "ScopeDefinition")
}

func (r *simpleStoreMapper) StoreForResource(resource schema.GroupResource) (string, error) {
	storeID, ok := r.overrides[resource]
	if ok {
		return storeID, nil
	}
	return r.defaultStoreID, nil
}

func (r *simpleStoreMapper) Stop() {
	r.stop()
}

var _ scopes.ResourceStoreMapper = &simpleStoreMapper{}
